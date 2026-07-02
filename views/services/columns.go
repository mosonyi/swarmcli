// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package servicesview

import (
	"fmt"

	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
)

// serviceColumn pairs a shared-layout column with its sort metadata. The set is
// the single source of truth for the header, widths, sort indices, and rows.
type serviceColumn struct {
	col      filterlist.Column[docker.ServiceEntry]
	sort     SortField
	hasSort  bool
	isStack  bool
	isHealth bool
}

// serviceColumns returns the columns for the current scope. The STACK column is
// dropped unless we're showing services from every stack (AllFilter), since in a
// single-stack/node scope every row carries the same stack value. The HEALTH
// column is dropped unless some row carries a health summary — ServiceEntry.Health
// is empty in the default (remote-API) loaders, so it is invisible unless a
// ServiceOps decorator populates it. Cell closures capture the model so the ERROR
// column can read the per-service error text.
func (m *Model) serviceColumns() []serviceColumn {
	all := []serviceColumn{
		{sort: SortByName, hasSort: true, col: filterlist.Column[docker.ServiceEntry]{
			Label: "SERVICE", MinWidth: 8, Flex: true,
			Cell: func(e docker.ServiceEntry) string { return e.ServiceName }}},
		{isStack: true, col: filterlist.Column[docker.ServiceEntry]{
			Label: "STACK", MinWidth: 6,
			Cell: func(e docker.ServiceEntry) string { return e.StackName }}},
		{col: filterlist.Column[docker.ServiceEntry]{
			Label: "REPLICAS", MinWidth: 5,
			Cell: func(e docker.ServiceEntry) string {
				if e.ReplicasTotal == 0 {
					return "—"
				}
				return fmt.Sprintf("%d/%d", e.ReplicasOnNode, e.ReplicasTotal)
			}}},
		{sort: SortByStatus, hasSort: true, col: filterlist.Column[docker.ServiceEntry]{
			Label: "STATUS", MinWidth: 8,
			Cell: func(e docker.ServiceEntry) string { return e.Status }}},
		{isHealth: true, col: filterlist.Column[docker.ServiceEntry]{
			Label: "HEALTH", MinWidth: 6,
			Cell: func(e docker.ServiceEntry) string {
				if e.Health == "" {
					return "—"
				}
				return e.Health
			}}},
		{col: filterlist.Column[docker.ServiceEntry]{
			Label: "MODE", MinWidth: 6,
			Cell: func(e docker.ServiceEntry) string { return e.Mode }}},
		{sort: SortByImage, hasSort: true, col: filterlist.Column[docker.ServiceEntry]{
			Label: "IMAGE", MinWidth: 8, Flex: true,
			Cell: func(e docker.ServiceEntry) string { return e.Image }}},
		{sort: SortByPorts, hasSort: true, col: filterlist.Column[docker.ServiceEntry]{
			Label: "PORTS", MinWidth: 5, Flex: true,
			Cell: func(e docker.ServiceEntry) string { return e.Ports }}},
		{sort: SortByCreated, hasSort: true, col: filterlist.Column[docker.ServiceEntry]{
			Label: "CREATED", MinWidth: 7,
			Cell: func(e docker.ServiceEntry) string { return formatRelativeTime(e.CreatedAt) }}},
		{sort: SortByUpdated, hasSort: true, col: filterlist.Column[docker.ServiceEntry]{
			Label: "UPDATED", MinWidth: 7,
			Cell: func(e docker.ServiceEntry) string { return formatRelativeTime(e.UpdatedAt) }}},
		{sort: SortByError, hasSort: true, col: filterlist.Column[docker.ServiceEntry]{
			Label: "ERROR", MinWidth: 6, Flex: true,
			Cell: func(e docker.ServiceEntry) string { return m.serviceErrorText[e.ServiceID] }}},
	}
	dropStack := m.filterType != AllFilter
	dropHealth := !m.anyServiceHealth()
	if !dropStack && !dropHealth {
		return all
	}
	out := make([]serviceColumn, 0, len(all))
	for _, c := range all {
		if c.isStack && dropStack {
			continue
		}
		if c.isHealth && dropHealth {
			continue
		}
		out = append(out, c)
	}
	return out
}

// anyServiceHealth reports whether any loaded service row carries a health
// summary, which gates the HEALTH column.
func (m *Model) anyServiceHealth() bool {
	for _, e := range m.List.Items {
		if e.Health != "" {
			return true
		}
	}
	return false
}

// layoutColumns extracts the shared-layout column set for the current scope.
func (m *Model) layoutColumns() []filterlist.Column[docker.ServiceEntry] {
	active := m.serviceColumns()
	cols := make([]filterlist.Column[docker.ServiceEntry], len(active))
	for i, c := range active {
		cols[i] = c.col
	}
	return cols
}

// sortIndicator reports which active column carries the current sort field and
// the sort direction, so the header arrow tracks the column even when STACK is
// dropped and indices shift.
func (m *Model) sortIndicator() (int, bool) {
	for i, c := range m.serviceColumns() {
		if c.hasSort && c.sort == m.sortField {
			return i, m.sortAscending
		}
	}
	return -1, true
}
