// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"fmt"
	"strconv"

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

const (
	revisionFormat = "    %-4s %-11s %-26s %-17s %s"
	serviceFormat  = "    %-26s %-12s %-10s %s"
)

// expansionBlock renders an expanded release's child lines and reports, for
// each selectable child, which line it landed on.
//
// The two are returned together on purpose: the scroll math needs a child's
// line index, and deriving it separately from a second count of headers and
// rows is how the two silently drift apart.
func expansionBlock(it releaseItem, selectedChild int) (lines []string, childLine []int) {
	rows := it.children()
	childLine = make([]int, len(rows))

	line := func(s string) { lines = append(lines, s) }
	style := func(idx int, s string) string {
		if idx == selectedChild {
			return childSelectedStyle.Render(s)
		}
		return childStyle.Render(s)
	}

	line(childHeaderStyle.Render(fmt.Sprintf(revisionFormat, "REV", "STATUS", "CHART", "UPDATED", "OWNER")))
	for n, rev := range it.Revisions {
		childLine[n] = len(lines)
		line(style(n, fmt.Sprintf(revisionFormat,
			strconv.Itoa(rev.Revision),
			truncate(rev.Status, 11),
			truncate(rev.Chart.Name+"-"+rev.Chart.Version, 26),
			formatCreated(parseCreated(rev.Created)),
			truncate(displayOrDash(rev.Owner), 30),
		)))
	}

	line(childHeaderStyle.Render(fmt.Sprintf(serviceFormat, "SERVICE", "MODE", "REPLICAS", "STATUS")))
	if len(it.Services) == 0 {
		line(childStyle.Render("    (no services)"))
		return lines, childLine
	}
	for n, svc := range it.Services {
		idx := len(it.Revisions) + n
		childLine[idx] = len(lines)
		line(style(idx, fmt.Sprintf(serviceFormat,
			truncate(svc.Name, 26),
			truncate(displayOrDash(svc.Mode), 12),
			truncate(displayOrDash(svc.Replicas), 10),
			displayOrDash(svc.Status),
		)))
	}
	return lines, childLine
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
