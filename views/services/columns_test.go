// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package servicesview

import (
	"testing"

	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

const longServiceName = "myproject_some-very-long-service-name"

// loadWithFilter pushes entries into the model under a given scope.
func loadWithFilter(m *Model, ft FilterType, entries []docker.ServiceEntry) {
	m.Update(Msg{Title: "T", Entries: entries, FilterType: ft})
}

func hasColumn(m *Model, label string) bool {
	return colLabelIndex(m, label) >= 0
}

func colLabelIndex(m *Model, label string) int {
	for i, c := range m.layoutColumns() {
		if c.Label == label {
			return i
		}
	}
	return -1
}

func TestColumns_StackOnlyInAllFilter(t *testing.T) {
	cases := []struct {
		ft        FilterType
		wantStack bool
		wantLen   int
	}{
		{AllFilter, true, 10},
		{StackFilter, false, 9},
		{NodeFilter, false, 9},
		{NoStackFilter, false, 9},
	}
	for _, tc := range cases {
		m := testModel()
		m.filterType = tc.ft
		require.Len(t, m.layoutColumns(), tc.wantLen, "filter %v", tc.ft)
		require.Equal(t, tc.wantStack, hasColumn(m, "STACK"), "filter %v stack presence", tc.ft)
	}
}

func TestColWidths_LengthMatchesHeader(t *testing.T) {
	for _, ft := range []FilterType{AllFilter, StackFilter, NodeFilter, NoStackFilter} {
		m := testModel()
		loadWithFilter(m, ft, fakeEntries("web", "api"))
		require.Len(t, m.List.ColWidths(), len(m.List.Header.Columns), "filter %v header sync", ft)
	}
}

func TestColWidths_WideTerminal_ServiceFits(t *testing.T) {
	m := testModel()
	loadWithFilter(m, AllFilter, fakeEntries(longServiceName, "api"))
	m.List.Viewport.Width = 200

	widths := m.List.ColWidths()
	svc := colLabelIndex(m, "SERVICE")
	// Column 0 renders with a leading space and a trailing gap, so its usable
	// content width is width-ColGap-1; it must fit the full name on a wide term.
	require.GreaterOrEqual(t, widths[svc]-filterlist.ColGap-1, len([]rune(longServiceName)),
		"SERVICE column must show the full name on a wide terminal")
}

func TestSortIndicator_ResolvesActiveIndex(t *testing.T) {
	// AllFilter: SERVICE,STACK,REPLICAS,STATUS,... -> STATUS at index 3.
	mAll := testModel()
	mAll.filterType = AllFilter
	mAll.sortField = SortByStatus
	idx, asc := mAll.sortIndicator()
	require.Equal(t, 3, idx)
	require.True(t, asc)

	// StackFilter drops STACK: SERVICE,REPLICAS,STATUS,... -> STATUS at index 2.
	mStack := testModel()
	mStack.filterType = StackFilter
	mStack.sortField = SortByStatus
	mStack.sortAscending = false
	idx, asc = mStack.sortIndicator()
	require.Equal(t, 2, idx)
	require.False(t, asc)
}

func TestView_FullServiceNameOnWideTerminal(t *testing.T) {
	m := testModel()
	loadWithFilter(m, AllFilter, fakeEntries(longServiceName, "api"))
	m.List.Viewport.Width = 200
	m.List.Viewport.Height = 20

	out := m.View()
	require.Contains(t, out, longServiceName, "full name must be visible, not truncated (#392)")
}

func TestView_FullServiceImageOnWideTerminal(t *testing.T) {
	const longImage = "registry.example.com/myproject/some-very-long-image-name:v1.2.3"
	m := testModel()
	entries := fakeEntries("web")
	entries[0].Image = longImage
	loadWithFilter(m, AllFilter, entries)
	m.List.Viewport.Width = 200
	m.List.Viewport.Height = 20

	out := m.View()
	require.Contains(t, out, longImage, "full image must be visible, not truncated")
}

func TestHeaderRowAlignment(t *testing.T) {
	// Header and rows must share the same total width across scopes and sort
	// fields, including the sorted column's " ▲" indicator (the #392 regression).
	for _, ft := range []FilterType{AllFilter, StackFilter} {
		for _, sf := range []SortField{SortByName, SortByStatus, SortByCreated, SortByError} {
			m := testModel()
			loadWithFilter(m, ft, fakeEntries(longServiceName, "api"))
			m.sortField = sf
			m.setRenderItem()
			for _, w := range []int{40, 120, 200} {
				m.List.Viewport.Width = w
				header := m.List.RenderHeader()
				row := m.List.RenderItem(m.List.Filtered[0], false, 0)
				require.Equal(t, lipgloss.Width(header), lipgloss.Width(row),
					"filter=%v sort=%v width=%d header/row width mismatch", ft, sf, w)
			}
		}
	}
}
