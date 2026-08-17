// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package filterlist

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Column is a generic, view-agnostic column descriptor for the content-aware
// table layout. A view declares its columns once; LayoutWidths and RenderRow
// derive widths, the header, and each row from the same set so they can never
// drift out of alignment.
type Column[T any] struct {
	Label    string // header label; also the no-truncate width floor
	MinWidth int    // declared content floor (excludes the inter-column gap)
	// Flex marks a column as elastic downwards: it gives up width first when the
	// terminal is too narrow, and horizontally scrolls when truncated on a
	// selected row.
	Flex bool
	// Grow marks the column that absorbs leftover width when the terminal is
	// wider than the content needs.
	//
	// It is separate from Flex because the two are not the same property, and a
	// table of short values makes that obvious: flexing NAME so a long one can
	// scroll on an 80-column terminal also hands it half the slack on a
	// 200-column one, opening a void in the middle of every row. Declaring Grow
	// on the trailing column instead puts the leftover after the last cell,
	// where it reads as margin.
	//
	// When no column declares Grow the Flex columns absorb the slack, which is
	// what every view did before Grow existed.
	Grow bool
	Cell func(T) string // extracts the plain cell text (the closure may capture model state)
}

// ColGap is the guaranteed minimum space between adjacent columns, baked into
// every column's width as trailing padding so it survives even when columns are
// shrunk to their floors on a narrow terminal.
const ColGap = 1

// displayWidth reports the rendered width of s in terminal cells, counting runes
// (not bytes) so measurement agrees with rune-based truncation on multibyte text.
func displayWidth(s string) int {
	return utf8.RuneCountInString(s)
}

// TruncateRunes truncates s to maxWidth runes, adding … if needed. It slices by
// runes so it never splits a multibyte character.
func TruncateRunes(s string, maxWidth int) string {
	r := []rune(s)
	if len(r) <= maxWidth {
		return s
	}
	if maxWidth <= 1 {
		return "…"
	}
	return string(r[:maxWidth-1]) + "…"
}

// ScrollWindow returns the window of full beginning at offset runes, truncated to
// maxWidth runes (with a trailing … when more remains to the right). Rune-based,
// so the offset and width never split a multibyte character.
func ScrollWindow(full string, offset, maxWidth int) string {
	if full == "" {
		return ""
	}
	r := []rune(full)
	if offset > len(r) {
		offset = len(r)
	}
	if offset < 0 {
		offset = 0
	}
	return TruncateRunes(string(r[offset:]), maxWidth)
}

// ColumnDefs builds header ColumnDefs (labels only) from a generic column set,
// keeping the header labels in lockstep with the layout columns.
func ColumnDefs[T any](cols []Column[T]) []ColumnDef {
	defs := make([]ColumnDef, len(cols))
	for i, c := range cols {
		defs[i] = ColumnDef{Label: c.Label}
	}
	return defs
}

// NaturalWidth is the width the table wants: every column at its content size,
// plus the gaps, with nothing stretched or squeezed. LayoutWidths returns
// exactly this when totalWidth equals it.
//
// A view uses it to decide whether a terminal has room to spare — for an extra
// column it only shows when there is genuine surplus, rather than one that
// squeezes every other column to fit.
func NaturalWidth[T any](cols []Column[T], items []T, sortCol int) int {
	n := len(cols)
	if n == 0 {
		return 0
	}
	sum := 0
	for i, c := range cols {
		sum += naturalColWidth(c, items, i == sortCol)
	}
	return sum + 1 + ColGap*n // +1 for column 0's leading space
}

// naturalColWidth is one column's content size: the widest of its label (plus
// the sort indicator when active), its declared minimum, and any cell.
func naturalColWidth[T any](c Column[T], items []T, sorted bool) int {
	fl := displayWidth(c.Label)
	if sorted {
		fl += 2 // " ▲"/" ▼"
	}
	if fl < c.MinWidth {
		fl = c.MinWidth
	}
	if fl < 3 {
		fl = 3
	}
	w := fl
	for _, it := range items {
		if cw := displayWidth(c.Cell(it)); cw > w {
			w = cw
		}
	}
	return w
}

// LayoutWidths sizes columns to their content so wide terminals no longer
// truncate while space sits empty, and guarantees at least one space between
// columns. sortCol is the index of the active sort column (-1 for none) so its
// floor reserves room for the header's " ▲"/" ▼" indicator. Each returned width
// includes the trailing gap; callers left-align text into it.
func LayoutWidths[T any](cols []Column[T], items []T, totalWidth, sortCol int) []int {
	if totalWidth <= 0 {
		totalWidth = 80
	}
	n := len(cols)
	if n == 0 {
		return nil
	}

	content := make([]int, n)
	floors := make([]int, n)
	flex := make([]bool, n)
	grow := make([]bool, n)
	anyGrow := false
	sum := 0
	for i, c := range cols {
		// Floor: the header label is never truncated by the renderer, so a column
		// must never shrink below its label width (plus the sort indicator on the
		// active sort column) or the header overflows and misaligns with the rows.
		// Also honour the column's declared minimum.
		fl := displayWidth(c.Label)
		if i == sortCol {
			fl += 2 // " ▲"/" ▼"
		}
		if fl < c.MinWidth {
			fl = c.MinWidth
		}
		if fl < 3 {
			fl = 3
		}

		// Natural content width: the widest of the floor and any cell.
		w := fl
		for _, it := range items {
			if cw := displayWidth(c.Cell(it)); cw > w {
				w = cw
			}
		}
		floors[i] = fl
		content[i] = w
		flex[i] = c.Flex
		grow[i] = c.Grow
		anyGrow = anyGrow || c.Grow
		sum += w
	}

	// Column 0 renders with a leading space (header convention); reserve one extra
	// cell in both its natural width and its floor so the leading space never eats
	// into the trailing gap.
	content[0]++
	floors[0]++
	sum++

	need := sum + ColGap*n
	switch {
	case need < totalWidth:
		// Hand the leftover to the Grow columns, falling back to the Flex ones
		// for a view that declares none — see Column.Grow.
		absorb := grow
		if !anyGrow {
			absorb = flex
		}
		distributeSlack(content, absorb, totalWidth-need)
	case need > totalWidth:
		shrinkColumns(content, floors, flex, need-totalWidth)
	}

	widths := make([]int, n)
	for i := range content {
		widths[i] = content[i] + ColGap
	}
	return widths
}

// RenderRow builds the plain (unstyled) row text for an item: each column is
// left-aligned into its width with a trailing gap; column 0 carries a leading
// space matching the header. A selected flex column whose content overflows is
// horizontally scrolled by scroll; otherwise content is rune-truncated.
func RenderRow[T any](cols []Column[T], widths []int, item T, scroll int, selected bool) string {
	if len(cols) == 0 {
		return ""
	}
	if len(widths) != len(cols) {
		return cols[0].Cell(item) // safe fallback; caller's width array is out of sync
	}
	cells := make([]string, len(cols))
	for i, c := range cols {
		raw := c.Cell(item)
		cw := widths[i] - ColGap
		lead := ""
		if i == 0 {
			lead = " "
			cw--
		}
		if cw < 1 {
			cw = 1
		}
		var text string
		if selected && c.Flex && displayWidth(raw) > cw {
			text = ScrollWindow(raw, scroll, cw)
		} else {
			text = TruncateRunes(raw, cw)
		}
		cells[i] = fmt.Sprintf("%s%-*s", lead, widths[i]-len(lead), text)
	}
	return strings.Join(cells, "")
}

// distributeSlack spreads leftover width across the flex columns, giving the
// rounding remainder to the first flex column so it grows first.
func distributeSlack(content []int, flex []bool, slack int) {
	if slack <= 0 {
		return
	}
	var idx []int
	for i, f := range flex {
		if f {
			idx = append(idx, i)
		}
	}
	if len(idx) == 0 {
		return
	}
	per := slack / len(idx)
	for _, i := range idx {
		content[i] += per
	}
	content[idx[0]] += slack % len(idx)
}

// shrinkColumns removes overflow width, taking from flex columns first and then
// non-flex columns, never below each column's floor so the header label still
// fits and the trailing gap always survives.
func shrinkColumns(content, floors []int, flex []bool, overflow int) {
	overflow = shrinkGroup(content, floors, flex, overflow, true)
	if overflow > 0 {
		shrinkGroup(content, floors, flex, overflow, false)
	}
}

// shrinkGroup reduces the columns whose flex flag matches flexOnly by up to
// overflow cells total, distributed proportionally to each column's shrinkable
// room. Returns the overflow it could not absorb.
func shrinkGroup(content, floors []int, flex []bool, overflow int, flexOnly bool) int {
	if overflow <= 0 {
		return 0
	}
	room := 0
	for i := range content {
		if flex[i] == flexOnly {
			room += content[i] - floors[i]
		}
	}
	if room <= 0 {
		return overflow
	}
	take := overflow
	if take > room {
		take = room
	}
	removed := 0
	for i := range content {
		if flex[i] != flexOnly {
			continue
		}
		r := content[i] - floors[i]
		if r <= 0 {
			continue
		}
		cut := take * r / room
		content[i] -= cut
		removed += cut
	}
	// Assign rounding leftover one cell at a time to columns with room.
	for leftover := take - removed; leftover > 0; {
		progressed := false
		for i := range content {
			if leftover == 0 {
				break
			}
			if flex[i] == flexOnly && content[i] > floors[i] {
				content[i]--
				leftover--
				progressed = true
			}
		}
		if !progressed {
			break
		}
	}
	return overflow - take
}
