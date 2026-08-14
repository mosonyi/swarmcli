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

// Widening the terminal must not push the columns apart.
//
// Release names and chart refs are short, so flexing them meant a 200-column
// terminal handed each half the leftover and opened a void in the middle of
// every row. Only the trailing column grows, so every column up to it sits at
// the same place whatever the width.
func TestWideTerminalDoesNotSpreadTheColumns(t *testing.T) {
	// Short values on purpose: that is when growing a middle column shows as a
	// void. Both widths are wider than the natural content, so nothing shrinks
	// and any difference is growth.
	short := map[string][]charts.Release{
		"openclaw": deployed("openclaw", "openclaw", "0.1.0"),
		"whoami":   deployed("whoami", "whoami", "0.1.9"),
	}
	positions := func(width int) map[string]int {
		m := testModel()
		m.list.Viewport.Width = width
		loadReleases(t, m, short, nil)
		header := m.list.RenderHeader()
		out := map[string]int{}
		for _, col := range []string{"REV", "STATUS", "HEALTH", "CHART", "LATEST"} {
			out[col] = columnOf(t, header, col)
		}
		return out
	}

	narrow := positions(100)
	wide := positions(240)
	require.Equal(t, narrow, wide,
		"a wider terminal must add margin after the last column, not between the others")

	// And the row still spans the frame, so the selection highlight does too.
	m := testModel()
	m.list.Viewport.Width = 240
	loadReleases(t, m, short, nil)
	require.Equal(t, 240, lipgloss.Width(m.list.RenderRow(m.list.Filtered[0], true)))
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
