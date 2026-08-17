// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package helpview

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// The cheat sheet's column count is a property of the terminal, never of how
// many categories an author wrote. Deriving it from the content — one column
// per category — is what made every description a candidate for truncation:
// each category added took space away from all the others, and the only
// defence left was to cut the sentence.

const (
	// minCategoryWidth is the narrowest a category column may be. Below it a
	// description is more wrapping than words, so the layout takes fewer,
	// wider columns instead.
	minCategoryWidth = 44

	// categoryGutter is the blank space kept between two columns.
	categoryGutter = 2

	// keyDescGap separates a key from its description.
	keyDescGap = 2

	// minDescWidth is the narrowest a description may be beside its key.
	// Under it the item stacks: key on its own line, description indented
	// below. The same fallback the per-command help screen uses for usage
	// lines, and for the same reason — a key that is really a phrase.
	minDescWidth = 16

	// stackedIndent indents a stacked item's description under its key.
	stackedIndent = 2
)

var (
	categoryTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("2"))
	itemKeyStyle       = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("33"))
	itemDescStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("252"))
)

// columnCount is how many category columns fit in width. At least one, never
// more than there are categories to put in them.
func columnCount(width, categories int) int {
	if categories < 1 {
		return 1
	}
	fit := width / minCategoryWidth
	if fit < 1 {
		fit = 1
	}
	if fit > categories {
		fit = categories
	}
	return fit
}

// renderCategory lays out one category as a block width columns wide. Every
// measurement here is display width — lipgloss wraps on it and never splits a
// rune, which is the other half of why this no longer cuts text.
func renderCategory(cat HelpCategory, width int) string {
	if width < 1 {
		width = 1
	}
	lines := []string{categoryTitleStyle.Render(strings.ToUpper(cat.Title))}

	keyWidth := 0
	for _, item := range cat.Items {
		if w := lipgloss.Width(item.Keys); w > keyWidth {
			keyWidth = w
		}
	}
	// A key column may not take more than half the block. Some categories
	// document values rather than keystrokes — a column name, a status word —
	// and one long entry must not leave every description in the category a
	// sliver.
	if half := width / 2; keyWidth > half {
		keyWidth = half
	}

	descWidth := width - keyWidth - keyDescGap
	for _, item := range cat.Items {
		lines = append(lines, renderItem(item, width, keyWidth, descWidth)...)
	}
	return strings.Join(lines, "\n")
}

// renderItem renders one key/description pair, side by side when the
// description has room to read and stacked when it does not.
func renderItem(item HelpItem, width, keyWidth, descWidth int) []string {
	key := itemKeyStyle.Render(item.Keys)
	if descWidth < minDescWidth {
		out := []string{key}
		if item.Description == "" {
			return out
		}
		indent := strings.Repeat(" ", stackedIndent)
		desc := itemDescStyle.Width(width - stackedIndent).Render(item.Description)
		for _, line := range strings.Split(desc, "\n") {
			out = append(out, indent+line)
		}
		return out
	}

	row := lipgloss.JoinHorizontal(lipgloss.Top,
		itemKeyStyle.Width(keyWidth).Render(item.Keys),
		strings.Repeat(" ", keyDescGap),
		itemDescStyle.Width(descWidth).Render(item.Description),
	)
	return strings.Split(row, "\n")
}

// packColumns distributes rendered category blocks over at most cols columns,
// keeping declaration order — a reader goes down a column and then on to the
// next, which is how the categories were written to be read.
//
// The split is the one that minimises the tallest column, found by binary
// search over the ceiling. Filling each column to an average and moving on does
// not work: whatever is left when the last column starts has to go in it, so
// the sheet ends with three categories stacked beside three empty ones.
func packColumns(blocks []string, cols int) []string {
	if cols < 1 {
		cols = 1
	}
	if cols >= len(blocks) {
		return blocks
	}

	heights := make([]int, len(blocks))
	lo, hi := 0, 0
	for i, b := range blocks {
		heights[i] = lipgloss.Height(b) + 1 // the blank line under the block
		if heights[i] > lo {
			lo = heights[i] // no column can be shorter than its tallest block
		}
		hi += heights[i]
	}
	for lo < hi {
		mid := (lo + hi) / 2
		if columnsNeeded(heights, mid) <= cols {
			hi = mid
		} else {
			lo = mid + 1
		}
	}
	return splitAt(blocks, heights, lo)
}

// columnsNeeded is how many columns the blocks take when none may exceed
// ceiling rows.
func columnsNeeded(heights []int, ceiling int) int {
	columns, current := 1, 0
	for _, h := range heights {
		if current > 0 && current+h > ceiling {
			columns++
			current = 0
		}
		current += h
	}
	return columns
}

// splitAt cuts the blocks into columns none of which exceeds ceiling rows.
func splitAt(blocks []string, heights []int, ceiling int) []string {
	var columns []string
	var current []string
	currentHeight := 0
	for i, b := range blocks {
		if currentHeight > 0 && currentHeight+heights[i] > ceiling {
			columns = append(columns, strings.Join(current, "\n\n"))
			current, currentHeight = nil, 0
		}
		current = append(current, b)
		currentHeight += heights[i]
	}
	if len(current) > 0 {
		columns = append(columns, strings.Join(current, "\n\n"))
	}
	return columns
}
