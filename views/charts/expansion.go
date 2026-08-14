// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"strconv"
	"strings"

	"github.com/Eldara-Tech/swarmcli/charts"

	"github.com/charmbracelet/lipgloss"
)

// noChild is the childIndex when the release row itself is selected.
const noChild = -1

type childKind int

const (
	childRevision childKind = iota
	childService
)

// childRow is one selectable line inside an expanded release. The column
// headers between the blocks are rendered but not selectable, so a child's
// index is not its line.
type childRow struct {
	kind childKind
	rev  charts.Release
	svc  charts.ServiceState
	// prev is the revision immediately below rev, for the diff. Its zero value
	// means rev is the first, which diffs against nothing.
	prev charts.Release
}

// children flattens an expanded release into its selectable rows: every stored
// revision ascending, then the live services.
func (i releaseItem) children() []childRow {
	rows := make([]childRow, 0, len(i.Revisions)+len(i.Services))
	for n, rev := range i.Revisions {
		row := childRow{kind: childRevision, rev: rev}
		if n > 0 {
			row.prev = i.Revisions[n-1]
		}
		rows = append(rows, row)
	}
	for _, svc := range i.Services {
		rows = append(rows, childRow{kind: childService, svc: svc})
	}
	return rows
}

var (
	childHeaderStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
	childStyle         = lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
	childSelectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24")).Bold(true)
)

// Child blocks are laid out at fixed widths rather than through the
// content-aware column layout, which sizes the release row above them.
var (
	revisionCols = []int{4, 11, 26, 17, 30}
	serviceCols  = []int{26, 12, 10, 0}
)

// childLine joins cells at fixed widths, padding by DISPLAY COLUMNS.
//
// fmt's "%-12s" pads by bytes, and these cells carry multi-byte runes as a
// matter of course — displayOrDash emits an em-dash for every empty optional
// field, which is three bytes and one column. Padding that by bytes shifts
// every later cell two columns left of its header.
func childLine(cells []string, widths []int) string {
	var b strings.Builder
	b.WriteString("    ")
	for i, cell := range cells {
		w := 0
		if i < len(widths) {
			w = widths[i]
		}
		if w == 0 { // last column: no padding, no truncation
			b.WriteString(cell)
			break
		}
		cell = truncate(cell, w)
		b.WriteString(cell)
		for pad := w - lipgloss.Width(cell); pad > 0; pad-- {
			b.WriteByte(' ')
		}
		b.WriteByte(' ')
	}
	return b.String()
}

// expansionBlock renders an expanded release's child lines and reports, for
// each selectable child, which line it landed on.
//
// The two are returned together on purpose: the scroll math needs a child's
// line index, and deriving it separately from a second count of headers and
// rows is how the two silently drift apart.
func expansionBlock(it releaseItem, selectedChild int) (lines []string, childLines []int) {
	rows := it.children()
	childLines = make([]int, len(rows))

	line := func(s string) { lines = append(lines, s) }
	style := func(idx int, s string) string {
		if idx == selectedChild {
			return childSelectedStyle.Render(s)
		}
		return childStyle.Render(s)
	}

	line(childHeaderStyle.Render(childLine(
		[]string{"REV", "STATUS", "CHART", "UPDATED", "OWNER"}, revisionCols)))
	for n, rev := range it.Revisions {
		childLines[n] = len(lines)
		line(style(n, childLine([]string{
			strconv.Itoa(rev.Revision),
			rev.Status,
			rev.Chart.Name + "-" + rev.Chart.Version,
			formatCreated(parseCreated(rev.Created)),
			displayOrDash(rev.Owner),
		}, revisionCols)))
	}

	line(childHeaderStyle.Render(childLine(
		[]string{"SERVICE", "MODE", "REPLICAS", "STATUS"}, serviceCols)))
	if len(it.Services) == 0 {
		line(childStyle.Render("    (no services)"))
		return lines, childLines
	}
	for n, svc := range it.Services {
		idx := len(it.Revisions) + n
		childLines[idx] = len(lines)
		line(style(idx, childLine([]string{
			svc.Name,
			displayOrDash(svc.Mode),
			displayOrDash(svc.Replicas),
			displayOrDash(svc.Status),
		}, serviceCols)))
	}
	return lines, childLines
}

// truncate shortens s to max display columns, with an ellipsis when it had to
// cut. Child blocks are laid out at fixed widths rather than through the
// content-aware column layout, which sizes the release row above them.
func truncate(s string, max int) string {
	if max <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	if max <= 3 {
		return string(r[:max])
	}
	return string(r[:max-3]) + "..."
}
