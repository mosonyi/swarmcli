// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"testing"

	"github.com/Eldara-Tech/swarmcli/charts"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

// longName overflows the NAME column on a narrow terminal and fits on a wide one.
const longName = "release_with_a_fairly_long_descriptive_name"

func twoReleases() map[string][]charts.Release {
	return map[string][]charts.Release{
		longName: deployed(longName, "some-long-chart-name", "1.2.3"),
		"z2":     deployed("z2", "c", "1.0.0"),
	}
}

func itemNamed(m *Model, name string) releaseItem {
	for _, r := range m.list.Filtered {
		if r.Name == name {
			return r
		}
	}
	return releaseItem{}
}

func TestHeaderAndRowAligned(t *testing.T) {
	for _, w := range []int{60, 120, 220} {
		m := testModel()
		m.list.Viewport.Width = w
		loadReleases(t, m, twoReleases(), nil)

		header := m.list.RenderHeader()
		row := m.list.RenderItem(m.list.Filtered[0], false, 0)
		require.Equal(t, lipgloss.Width(header), lipgloss.Width(row),
			"header and row widths must match at width %d", w)
	}
}

func TestContentAwareNameWideTerminal(t *testing.T) {
	m := testModel()
	m.list.Viewport.Width = 220
	loadReleases(t, m, twoReleases(), nil)
	row := m.list.RenderItem(itemNamed(m, longName), false, 0)
	require.Contains(t, row, longName, "full name must be visible on a wide terminal")
}

func TestArrowScrollMovesWindow(t *testing.T) {
	m := testModel()
	m.list.Viewport.Width = 70 // narrow → flex columns overflow
	loadReleases(t, m, twoReleases(), nil)
	before := m.list.RenderItem(itemNamed(m, longName), true, 0)
	m.Update(key("right"))
	after := m.list.RenderItem(itemNamed(m, longName), true, 0)
	require.NotEqual(t, before, after, "right arrow should shift the truncated cell window")
}

func TestResetScrollOnCursorMove(t *testing.T) {
	m := testModel()
	m.list.Viewport.Width = 70
	loadReleases(t, m, twoReleases(), nil)
	m.Update(key("right"))
	m.Update(key("right"))
	scrolled := m.list.RenderItem(itemNamed(m, longName), true, 0)

	m.Update(key("down"))
	m.Update(key("up"))

	require.NotEqual(t, scrolled, m.list.RenderItem(itemNamed(m, longName), true, 0),
		"scroll offset must reset when the cursor moves")
}

// The sort arrow must land on the column the sort field names, or the header
// tells the operator something false.
func TestSortIndicatorTracksTheSortField(t *testing.T) {
	m := testModel()
	cols := m.buildColumns()
	for field, want := range map[SortField]string{
		SortByName:     "NAME",
		SortByRevision: "REV",
		SortByStatus:   "STATUS",
		SortByHealth:   "HEALTH",
		SortByChart:    "CHART",
		SortByUpdated:  "UPDATED",
	} {
		m.sortField = field
		require.Equal(t, want, cols[m.sortColumnIndex()].Label)
	}
}
