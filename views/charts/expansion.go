// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"strconv"

	"github.com/Eldara-Tech/swarmcli/charts"
	filterlist "github.com/Eldara-Tech/swarmcli/ui/components/filterable/list"

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

// childIndent is the space the child block is inset by, so the expansion reads
// as belonging to the release above it.
const childIndent = "    "

// revisionColumns and serviceColumns describe the two child blocks.
//
// They go through the same content-aware layout as the release row rather than
// carrying hardcoded widths. Fixed widths overflowed a narrow frame — the OWNER
// stamp ran into the border and was cut mid-word by the frame rather than
// truncated — and they padded by bytes, so an em-dash shifted every later cell
// two columns off its header.
func revisionColumns() []filterlist.Column[charts.Release] {
	return []filterlist.Column[charts.Release]{
		{Label: "REV", MinWidth: 3, Cell: func(r charts.Release) string { return strconv.Itoa(r.Revision) }},
		{Label: "STATUS", MinWidth: 6, Cell: func(r charts.Release) string { return displayOrDash(r.Status) }},
		{Label: "CHART", MinWidth: 5, Flex: true, Cell: func(r charts.Release) string {
			return r.Chart.Name + "-" + r.Chart.Version
		}},
		{Label: "UPDATED", MinWidth: 16, Cell: func(r charts.Release) string { return formatCreated(parseCreated(r.Created)) }},
		{Label: "OWNER", MinWidth: 5, Flex: true, Grow: true, Cell: func(r charts.Release) string {
			return ownerCell(r.Owner, r.Name)
		}},
	}
}

func serviceColumns() []filterlist.Column[charts.ServiceState] {
	return []filterlist.Column[charts.ServiceState]{
		{Label: "SERVICE", MinWidth: 7, Flex: true, Cell: func(s charts.ServiceState) string { return s.Name }},
		{Label: "MODE", MinWidth: 4, Cell: func(s charts.ServiceState) string { return displayOrDash(s.Mode) }},
		{Label: "REPLICAS", MinWidth: 8, Cell: func(s charts.ServiceState) string { return displayOrDash(s.Replicas) }},
		{Label: "STATUS", MinWidth: 6, Flex: true, Grow: true, Cell: func(s charts.ServiceState) string {
			return displayOrDash(s.Status)
		}},
	}
}

// childWidth is the width a child row has to work with: the frame's content
// area less the indent. Zero or less means "unknown", and the block falls back
// to its natural size rather than collapsing to nothing.
func childWidth(total int) int {
	w := total - len(childIndent)
	if w < 20 {
		return 0
	}
	return w
}

// expansionBlock renders an expanded release's child lines and reports, for
// each selectable child, which line it landed on.
//
// The two are returned together on purpose: the scroll math needs a child's
// line index, and deriving it separately from a second count of headers and
// rows is how the two silently drift apart.
func expansionBlock(it releaseItem, selectedChild, width int) (lines []string, childLines []int) {
	rows := it.children()
	childLines = make([]int, len(rows))

	line := func(s string) { lines = append(lines, s) }
	style := func(idx int, s string) string {
		if idx == selectedChild {
			return childSelectedStyle.Render(s)
		}
		return childStyle.Render(s)
	}

	w := childWidth(width)
	revCols := revisionColumns()
	revWidths := layoutOrNatural(revCols, it.Revisions, w)
	line(childHeaderStyle.Render(childIndent + headerRow(revCols, revWidths)))
	for n, rev := range it.Revisions {
		childLines[n] = len(lines)
		line(style(n, childIndent+filterlist.RenderRow(revCols, revWidths, rev, 0, false)))
	}

	svcCols := serviceColumns()
	svcWidths := layoutOrNatural(svcCols, it.Services, w)
	line(childHeaderStyle.Render(childIndent + headerRow(svcCols, svcWidths)))
	if len(it.Services) == 0 {
		line(childStyle.Render(childIndent + "(no services)"))
		return lines, childLines
	}
	for n, svc := range it.Services {
		idx := len(it.Revisions) + n
		childLines[idx] = len(lines)
		line(style(idx, childIndent+filterlist.RenderRow(svcCols, svcWidths, svc, 0, false)))
	}
	return lines, childLines
}

// headerRow renders the column labels through the very same RenderRow the data
// rows use, by swapping each Cell for its own Label. Alignment between a child
// header and its rows is then true by construction rather than by two pieces of
// formatting code agreeing.
func headerRow[T any](cols []filterlist.Column[T], widths []int) string {
	hdr := make([]filterlist.Column[T], len(cols))
	for i, c := range cols {
		label := c.Label
		hdr[i] = filterlist.Column[T]{Label: c.Label, MinWidth: c.MinWidth, Cell: func(T) string { return label }}
	}
	var zero T
	return filterlist.RenderRow(hdr, widths, zero, 0, false)
}

// layoutOrNatural sizes the child columns to the width available, or to their
// content when the width is unknown.
func layoutOrNatural[T any](cols []filterlist.Column[T], items []T, width int) []int {
	if width <= 0 {
		width = filterlist.NaturalWidth(cols, items, -1)
	}
	return filterlist.LayoutWidths(cols, items, width, -1)
}
