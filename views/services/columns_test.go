// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package servicesview

import (
	"strings"
	"testing"

	"swarmcli/docker"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

const longServiceName = "myproject_some-very-long-service-name"

// loadWithFilter pushes entries into the model under a given scope.
func loadWithFilter(m *Model, ft FilterType, entries []docker.ServiceEntry) {
	m.Update(Msg{Title: "T", Entries: entries, FilterType: ft})
}

// colIndex returns the position of a column id in the active set, or -1.
func colIndex(m *Model, id colID) int {
	for i, spec := range m.activeColumns() {
		if spec.id == id {
			return i
		}
	}
	return -1
}

func TestActiveColumns_StackOnlyInAllFilter(t *testing.T) {
	cases := []struct {
		ft         FilterType
		wantStack  bool
		wantLength int
	}{
		{AllFilter, true, len(servicesColumnTemplate)},
		{StackFilter, false, len(servicesColumnTemplate) - 1},
		{NodeFilter, false, len(servicesColumnTemplate) - 1},
		{NoStackFilter, false, len(servicesColumnTemplate) - 1},
	}
	for _, tc := range cases {
		m := testModel()
		m.filterType = tc.ft
		active := m.activeColumns()
		require.Len(t, active, tc.wantLength, "filter %v", tc.ft)
		require.Equal(t, tc.wantStack, colIndex(m, colStack) >= 0, "filter %v stack presence", tc.ft)
	}
}

func TestComputeColWidths_LengthMatchesHeader(t *testing.T) {
	for _, ft := range []FilterType{AllFilter, StackFilter, NodeFilter, NoStackFilter} {
		m := testModel()
		loadWithFilter(m, ft, fakeEntries("web", "api"))
		widths := m.computeColWidths(120)
		require.Len(t, widths, len(m.activeColumns()), "filter %v", ft)
		require.Len(t, widths, len(m.List.Header.Columns), "filter %v header sync", ft)
	}
}

func TestComputeColWidths_WideTerminal_ServiceFits(t *testing.T) {
	m := testModel()
	loadWithFilter(m, AllFilter, fakeEntries(longServiceName, "api"))

	width := 200
	widths := m.computeColWidths(width)

	// Column 0 renders with a leading space and a trailing gap, so its usable
	// content width is colWidths[0]-colGap-1. It must fit the full name.
	svc := colIndex(m, colService)
	content0 := widths[svc] - colGap - 1
	require.GreaterOrEqual(t, content0, displayWidth(longServiceName),
		"SERVICE column must show the full name on a wide terminal")

	sum := 0
	for _, w := range widths {
		sum += w
	}
	require.LessOrEqual(t, sum, width, "columns must fit the viewport when content is small")
}

func TestComputeColWidths_GapPreservedWhenNarrow(t *testing.T) {
	m := testModel()
	long := fakeEntries(longServiceName, "another-fairly-long-service-name")
	long[0].Image = "registry.example.com/team/very/long/image/path:tag"
	long[0].Ports = "8080:8080,9090:9090,7070:7070"
	loadWithFilter(m, AllFilter, long)

	widths := m.computeColWidths(40)
	active := m.activeColumns()

	// Even fully truncated, every column keeps room for its (untruncated) header
	// label plus the inter-column gap, so labels stay readable, the header can't
	// overflow, and columns never merge.
	for i, spec := range active {
		require.GreaterOrEqual(t, widths[i]-colGap, displayWidth(spec.label),
			"column %q must retain at least its label width", spec.label)
	}
}

func TestComputeColWidths_EmptyAndZeroWidth(t *testing.T) {
	m := testModel()
	loadWithFilter(m, AllFilter, nil) // empty list

	require.NotPanics(t, func() {
		w := m.computeColWidths(0) // zero width behaves as 80
		require.Len(t, w, len(m.activeColumns()))
		for i, spec := range m.activeColumns() {
			require.GreaterOrEqual(t, w[i], spec.minWidth, "column %q at least label/min width", spec.label)
		}
	})
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
	m.setRenderItem()
	m.List.Viewport.Width = 200
	m.List.Viewport.Height = 20

	out := m.View()
	require.Contains(t, out, longServiceName, "full name must be visible, not truncated (#392)")
}

func TestHeaderRowAlignment(t *testing.T) {
	// Header and rows must share the same total width (sum of colWidths) across
	// scopes and sort fields, including the sorted column's " ▲" indicator.
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

func TestTruncateWithEllipsis_RuneAware(t *testing.T) {
	require.Equal(t, "héllo", truncateWithEllipsis("héllo", 5))
	require.Equal(t, "hé…", truncateWithEllipsis("héllo", 3))
	require.Equal(t, "…", truncateWithEllipsis("héllo", 1))
	// Byte-based slicing would have split the multibyte rune; rune-based must not.
	require.True(t, strings.HasSuffix(truncateWithEllipsis("héllo", 3), "…"))
}
