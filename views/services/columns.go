// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package servicesview

import (
	"fmt"
	"unicode/utf8"

	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
)

// colID identifies a logical services column independent of its position in the
// active column set (the STACK column is dropped outside the All-Services scope).
type colID int

const (
	colService colID = iota
	colStack
	colReplicas
	colStatus
	colMode
	colImage
	colPorts
	colCreated
	colUpdated
	colError
)

// colSpec is the single source of truth for a services column: its header label,
// width policy, sort mapping, and how to extract its cell text from an entry.
// Header labels, sort indices, column widths, and row cells all derive from the
// active []colSpec so they can never drift out of sync.
type colSpec struct {
	id       colID
	label    string
	minWidth int  // content floor (excludes the inter-column gap)
	flex     bool // absorbs leftover slack; horizontally scrolls when truncated on a selected row
	sort     SortField
	hasSort  bool
	cell     func(m *Model, e docker.ServiceEntry) string
}

// servicesColumnTemplate is the canonical, ordered set of all services columns.
// activeColumns derives the per-scope subset from it.
var servicesColumnTemplate = []colSpec{
	{id: colService, label: "SERVICE", minWidth: 8, flex: true, sort: SortByName, hasSort: true,
		cell: func(_ *Model, e docker.ServiceEntry) string { return e.ServiceName }},
	{id: colStack, label: "STACK", minWidth: 6,
		cell: func(_ *Model, e docker.ServiceEntry) string { return e.StackName }},
	{id: colReplicas, label: "REPLICAS", minWidth: 5,
		cell: func(_ *Model, e docker.ServiceEntry) string {
			if e.ReplicasTotal == 0 {
				return "—"
			}
			return fmt.Sprintf("%d/%d", e.ReplicasOnNode, e.ReplicasTotal)
		}},
	{id: colStatus, label: "STATUS", minWidth: 8, sort: SortByStatus, hasSort: true,
		cell: func(_ *Model, e docker.ServiceEntry) string { return e.Status }},
	{id: colMode, label: "MODE", minWidth: 6,
		cell: func(_ *Model, e docker.ServiceEntry) string { return e.Mode }},
	{id: colImage, label: "IMAGE", minWidth: 8, flex: true, sort: SortByImage, hasSort: true,
		cell: func(_ *Model, e docker.ServiceEntry) string { return e.Image }},
	{id: colPorts, label: "PORTS", minWidth: 5, flex: true, sort: SortByPorts, hasSort: true,
		cell: func(_ *Model, e docker.ServiceEntry) string { return e.Ports }},
	{id: colCreated, label: "CREATED", minWidth: 7, sort: SortByCreated, hasSort: true,
		cell: func(_ *Model, e docker.ServiceEntry) string { return formatRelativeTime(e.CreatedAt) }},
	{id: colUpdated, label: "UPDATED", minWidth: 7, sort: SortByUpdated, hasSort: true,
		cell: func(_ *Model, e docker.ServiceEntry) string { return formatRelativeTime(e.UpdatedAt) }},
	{id: colError, label: "ERROR", minWidth: 6, flex: true, sort: SortByError, hasSort: true,
		cell: func(m *Model, e docker.ServiceEntry) string { return m.serviceErrorText[e.ServiceID] }},
}

// activeColumns returns the columns to render for the current scope. The STACK
// column is dropped unless we're showing services from every stack (AllFilter),
// since in a single-stack/node scope every row carries the same stack value.
func (m *Model) activeColumns() []colSpec {
	if m.filterType == AllFilter {
		return servicesColumnTemplate
	}
	return without(servicesColumnTemplate, colStack)
}

// activeColumnLabels returns just the labels of the active columns, for building
// the filterable-list header.
func (m *Model) activeColumnLabels() []string {
	active := m.activeColumns()
	labels := make([]string, len(active))
	for i, c := range active {
		labels[i] = c.label
	}
	return labels
}

// headerColumns builds the filterable-list header column set from labels.
func headerColumns(labels []string) []filterlist.ColumnDef {
	cols := make([]filterlist.ColumnDef, len(labels))
	for i, l := range labels {
		cols[i] = filterlist.ColumnDef{Label: l}
	}
	return cols
}

// sortIndicator reports which active column carries the current sort field and
// the sort direction, so the header arrow tracks the column even when STACK is
// dropped and indices shift.
func (m *Model) sortIndicator() (int, bool) {
	for i, spec := range m.activeColumns() {
		if spec.hasSort && spec.sort == m.sortField {
			return i, m.sortAscending
		}
	}
	return -1, true
}

func without(specs []colSpec, id colID) []colSpec {
	out := make([]colSpec, 0, len(specs))
	for _, s := range specs {
		if s.id != id {
			out = append(out, s)
		}
	}
	return out
}

// colGap is the guaranteed minimum space between adjacent columns. It is baked
// into every column's returned width as trailing padding, so the gap survives
// even when columns are shrunk to their floors on a narrow terminal.
const colGap = 1

// distributeSlack spreads leftover width across the flex columns, giving the
// rounding remainder to the first flex column (SERVICE) so names grow first.
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

// displayWidth reports the rendered width of s in terminal cells. Service and
// image names are ASCII in practice, but counting runes (not bytes) keeps width
// measurement consistent with truncateWithEllipsis for any multibyte content.
func displayWidth(s string) int {
	return utf8.RuneCountInString(s)
}
