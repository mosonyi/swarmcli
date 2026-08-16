// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"testing"
	"time"

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

// The OWNER cell names the tool that installed the release, and drops the half
// of the stamp that only repeats the row it is drawn on.
func TestOwnerColumnNamesTheControllerAndDropsTheResourceHalf(t *testing.T) {
	m := sized(testModel(), 200, 24)
	owned := rev("whoami", 1, charts.StatusDeployed, "whoami", "0.1.9")
	owned.Owner = "cd/default/whoami:release/whoami"
	loadReleases(t, m, map[string][]charts.Release{"whoami": {owned}},
		map[string][]charts.ServiceState{"whoami": {converged("whoami_web")}})

	row := m.list.RenderItem(m.list.Filtered[0], false, 0)
	require.Contains(t, row, "swarmcli-cd/default/whoami")
	require.NotContains(t, row, ":release/whoami", "the resource half is the NAME column again")
}

// An owner wider than its column is reachable with ←/→, and stays reachable:
// the arrow stops where the text ends instead of scrolling the cell off it.
func TestArrowScrollReachesTheOwnerAndStopsAtItsEnd(t *testing.T) {
	// Wide enough for the OWNER column, too narrow for this owner to fit in it.
	m := sized(testModel(), 130, 24)
	owned := rev("app", 1, charts.StatusDeployed, "c", "1.0.0")
	owned.Owner = "cd/prod-cluster/eldara-renovate-and-friends:release/app"
	loadReleases(t, m, map[string][]charts.Release{"app": {owned}},
		map[string][]charts.ServiceState{"app": {converged("app_web")}})

	require.NotContains(t, m.list.RenderItem(m.list.Filtered[0], true, 0), "and-friends",
		"the owner must start out truncated, or this proves nothing")

	for range 20 {
		m.Update(key("right"))
	}
	require.Contains(t, m.list.RenderItem(m.list.Filtered[0], true, 0), "and-friends",
		"the arrow reaches the end of the owner, and stops there rather than past it")
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
		m := sized(testModel(), width, 24)
		loadReleases(t, m, short, nil)
		header := m.list.RenderHeader()
		out := map[string]int{}
		for _, col := range []string{"REV", "STATUS", "HEALTH", "CHART", "LATEST"} {
			out[col] = columnOf(t, header, col)
		}
		return out
	}

	// Both widths carry the same column set (short values, so DETAIL and OWNER
	// are affordable at each), and the columns before the growing one must not
	// drift apart between them.
	narrow := positions(150)
	wide := positions(240)
	require.Equal(t, narrow, wide,
		"a wider terminal must feed the growing column, not spread the others")

	// And the row still spans the frame, so the selection highlight does too.
	m := sized(testModel(), 240, 24)
	loadReleases(t, m, short, nil)
	require.Equal(t, 240, lipgloss.Width(m.list.RenderRow(m.list.Filtered[0], true)),
		"and the row still spans the frame, so the selection highlight does too")
}

// Surplus width buys information, not air.
//
// Two earlier attempts were wrong in opposite directions: sharing the slack
// between NAME and CHART opened voids in the middle of every row, and giving it
// to no one left half a wide terminal dead. The column set is responsive
// instead — each tier of width adds a column that has something to say.
func TestWiderTerminalsAddColumnsRatherThanPadding(t *testing.T) {
	labels := func(width int) []string {
		m := sized(testModel(), width, 24)
		loadReleases(t, m,
			map[string][]charts.Release{"app": deployed("app", "c", "1.0.0")},
			map[string][]charts.ServiceState{"app": {converged("app_web")}})
		out := make([]string, 0, len(m.list.Columns))
		for _, c := range m.list.Columns {
			out = append(out, c.Label)
		}
		return out
	}

	base := []string{"NAME", "REV", "STATUS", "HEALTH", "CHART", "LATEST", "UPDATED"}
	require.Equal(t, base, labels(80), "a narrow terminal shows the compact set")
	require.Equal(t, append(append([]string{}, base...), "DETAIL"), labels(120))
	require.Equal(t, append(append([]string{}, base...), "OWNER", "DETAIL"), labels(190))
}

// The extra column must be earned, never taken from the others: adding DETAIL
// by squeezing NAME and CHART would trade a readable name for a truncated
// explanation.
func TestTheDetailColumnNeverSqueezesTheOthers(t *testing.T) {
	long := map[string][]charts.Release{
		"a-release-with-a-really-quite-long-name": deployed("a-release-with-a-really-quite-long-name", "some-long-chart-name", "1.2.3"),
	}

	withoutDetail := sized(testModel(), 100, 24)
	loadReleases(t, withoutDetail, long, nil)
	require.False(t, withoutDetail.hasDetailColumn(),
		"the long name eats the surplus, so DETAIL must stand down")

	// Widen until it is affordable; the name must still be intact.
	withDetail := sized(testModel(), 190, 24)
	loadReleases(t, withDetail, long, nil)
	require.True(t, withDetail.hasDetailColumn())
	require.Contains(t, withDetail.View(), "a-release-with-a-really-quite-long-name",
		"DETAIL was added out of surplus, not out of NAME")
}

// DETAIL is the one column allowed to give way, so a terminal between the tiers
// truncates the explanation rather than the identity.
func TestOnlyDetailTruncatesWhenSpaceIsTight(t *testing.T) {
	m := sized(testModel(), 118, 24)
	loadReleases(t, m,
		map[string][]charts.Release{"app": deployed("app", "chartname", "1.0.0")},
		map[string][]charts.ServiceState{"app": {{
			Name: "app_web", Running: 1, Desired: 4, NewestTaskAge: time.Hour,
		}}})
	require.True(t, m.hasDetailColumn())

	out := m.View()
	require.Contains(t, out, "chartname-1.0.0", "the chart ref is intact")
	require.Contains(t, out, "app", "the name is intact")
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
