// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package servicesview

import (
	"strings"
	"testing"

	"swarmcli/docker"
	"swarmcli/features"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/view"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

// healthCell returns the rendered HEALTH cell for e, or "" when the HEALTH
// column is not in the current column set.
func healthCell(m *Model, e docker.ServiceEntry) string {
	for _, c := range m.serviceColumns() {
		if c.isHealth {
			return c.col.Cell(e)
		}
	}
	return ""
}

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
	// CE default (feature off): the HEALTH column stays visible via its footnote,
	// so it is present in every scope; only STACK varies with the filter.
	features.Disable(serviceHealthFeature)
	view.ServicesHealthHint = nil
	cases := []struct {
		ft        FilterType
		wantStack bool
		wantLen   int
	}{
		{AllFilter, true, 11},
		{StackFilter, false, 10},
		{NodeFilter, false, 10},
		{NoStackFilter, false, 10},
	}
	for _, tc := range cases {
		m := testModel()
		m.filterType = tc.ft
		require.Len(t, m.layoutColumns(), tc.wantLen, "filter %v", tc.ft)
		require.Equal(t, tc.wantStack, hasColumn(m, "STACK"), "filter %v stack presence", tc.ft)
	}
}

func TestColumns_HealthColumnAndFootnote(t *testing.T) {
	// Feature off (CE / unlicensed): a footnote explains the column, so HEALTH is
	// shown with a "*" placeholder rather than dropped.
	features.Disable(serviceHealthFeature)
	view.ServicesHealthHint = nil
	m := testModel()
	loadWithFilter(m, AllFilter, fakeEntries("web", "api"))
	require.True(t, hasColumn(m, "HEALTH"), "HEALTH stays visible with a footnote")
	require.Equal(t, "*", healthCell(m, docker.ServiceEntry{}), "no-health cell shows the footnote asterisk")

	// Feature on + reachable (empty note), no row has health → HEALTH is dropped.
	features.Enable(serviceHealthFeature)
	t.Cleanup(func() { features.Disable(serviceHealthFeature) })
	view.ServicesHealthHint = func() string { return "" }
	t.Cleanup(func() { view.ServicesHealthHint = nil })
	m2 := testModel()
	loadWithFilter(m2, AllFilter, fakeEntries("web", "api"))
	require.False(t, hasColumn(m2, "HEALTH"), "HEALTH hidden when reachable and no row has health")

	// A populated Health summary shows the real value, right after STATUS.
	entries := fakeEntries("web", "api")
	entries[0].Health = "1/1 healthy"
	m3 := testModel()
	loadWithFilter(m3, AllFilter, entries)
	require.True(t, hasColumn(m3, "HEALTH"), "HEALTH appears when a row has health")
	require.Equal(t, "1/1 healthy", healthCell(m3, docker.ServiceEntry{Health: "1/1 healthy"}))
	require.Equal(t, colLabelIndex(m3, "STATUS")+1, colLabelIndex(m3, "HEALTH"),
		"HEALTH should sit immediately after STATUS")
}

func TestExpandedRow_ContainerStateFallback(t *testing.T) {
	m := testModel()
	loadWithFilter(m, AllFilter, fakeEntries("web"))
	// Expand the service and give it a replica that is erroring with no
	// healthcheck (Health empty). The container's live state must still surface
	// as the HEALTH fallback so failures show for images without a HEALTHCHECK.
	m.expandedServices["id-web"] = true
	m.serviceTasks["id-web"] = []docker.TaskEntry{{
		Name: "web.1", NodeName: "node-1", DesiredState: "Running",
		CurrentState: "running 8m", ContainerID: "c1", ContainerState: "exited",
	}}
	m.setRenderItem()

	out := m.List.RenderItem(m.List.Filtered[0], false, 0)
	require.Contains(t, out, "HEALTH", "HEALTH column must appear when a replica carries container state")
	require.Contains(t, out, "exited", "the container's live state must render as the HEALTH fallback")
}

func TestTaskRowStyle_TintsFailureStates(t *testing.T) {
	red, yellow, grey := lipgloss.Color("9"), lipgloss.Color("3"), lipgloss.Color("7")
	cases := map[string]lipgloss.Color{
		"unhealthy":  red,
		"exited":     red,
		"dead":       red,
		"starting":   yellow,
		"restarting": yellow,
		"healthy":    grey,
		"running":    grey,
		"":           grey,
	}
	for status, want := range cases {
		require.Equal(t, want, taskRowStyle(status).GetForeground(), "status %q", status)
	}
}

func TestFirstNonEmpty(t *testing.T) {
	require.Equal(t, "healthy", firstNonEmpty("healthy", "running"))
	require.Equal(t, "exited", firstNonEmpty("", "exited"))
	require.Equal(t, "", firstNonEmpty("", ""))
}

func TestFormatTaskRow_ConditionalColumns(t *testing.T) {
	// Neither flag set → layout matches the pre-existing NAME…ERROR row.
	plain := formatTaskRow("web.1", "node-1", "Running", "running 8m", "healthy", "8000/tcp", "boom", false, false)
	require.NotContains(t, plain, "healthy")
	require.NotContains(t, plain, "8000/tcp")
	require.Contains(t, plain, "boom")

	// Both flags set → HEALTH and PORTS appear before ERROR.
	full := formatTaskRow("web.1", "node-1", "Running", "running 8m", "healthy", "8000/tcp", "boom", true, true)
	require.Contains(t, full, "healthy")
	require.Contains(t, full, "8000/tcp")
	require.Less(t, indexOf(full, "healthy"), indexOf(full, "boom"))
	require.Less(t, indexOf(full, "8000/tcp"), indexOf(full, "boom"))
}

func indexOf(s, sub string) int {
	return strings.Index(s, sub)
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
