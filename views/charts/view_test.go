// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"strings"
	"testing"
	"time"

	"github.com/Eldara-Tech/swarmcli/charts"

	"github.com/stretchr/testify/require"
)

func sized(m *Model, w, h int) *Model {
	m.SetSize(w, h)
	return m
}

func TestViewLoadingState(t *testing.T) {
	m := sized(testModel(), 120, 24)
	require.Equal(t, stateLoading, m.state)
	out := m.View()
	require.Contains(t, out, "Chart Releases")
	require.Contains(t, out, "Loading")
}

func TestViewReadyStateShowsReleases(t *testing.T) {
	m := sized(testModel(), 120, 24)
	loadReleases(t, m,
		map[string][]charts.Release{
			"edge":  deployed("edge", "traefik", "0.1.1"),
			"hello": deployed("hello", "whoami", "0.1.8"),
		},
		map[string][]charts.ServiceState{"edge": {converged("edge_web")}})

	out := m.View()
	require.Contains(t, out, "Chart Releases (2)")
	require.Contains(t, out, "edge")
	require.Contains(t, out, "hello")
	require.Contains(t, out, "traefik-0.1.1")
	require.Contains(t, out, "converged")
}

func TestHeaderShowsAllColumns(t *testing.T) {
	m := sized(testModel(), 140, 24)
	loadReleases(t, m, map[string][]charts.Release{"app": deployed("app", "c", "1.0.0")}, nil)

	header := m.FrameHeader()
	for _, col := range []string{"NAME", "REV", "STATUS", "HEALTH", "CHART", "UPDATED"} {
		require.Contains(t, header, col)
	}
}

// STATUS and HEALTH are both present on purpose, and a release that reads
// deployed while its rollout is stuck is exactly why.
func TestBothStatusAndHealthAreRendered(t *testing.T) {
	m := sized(testModel(), 140, 24)
	loadReleases(t, m,
		map[string][]charts.Release{"app": deployed("app", "c", "1.0.0")},
		map[string][]charts.ServiceState{"app": {{
			Name: "app_web", Running: 1, Desired: 1,
			UpdateState: "paused", NewestTaskAge: time.Hour,
		}}})

	out := m.View()
	require.Contains(t, out, "deployed", "the stored record")
	require.Contains(t, out, "wedged", "what the swarm is actually doing")
}

// Charts are opt-in, so an empty list must not read as a broken view.
func TestEmptyStateNamesTheInstallCommand(t *testing.T) {
	m := sized(testModel(), 120, 24)
	loadReleases(t, m, nil, nil)

	out := m.View()
	require.Contains(t, out, "No chart releases found")
	require.Contains(t, out, "swarmcli charts install")
}

// An active filter that matches nothing is a different situation from having
// no releases at all, and must not advertise the install command.
func TestFilteredToNothingIsNotTheEmptyState(t *testing.T) {
	m := sized(testModel(), 120, 24)
	loadReleases(t, m, map[string][]charts.Release{"app": deployed("app", "c", "1.0.0")}, nil)
	m.ApplySearchQuery("nothing-matches-this")

	out := m.View()
	require.NotContains(t, out, "No chart releases found",
		"there are releases; the filter simply matched none of them")
	require.Contains(t, out, "No releases")
}

func TestFooterShowsCountsAndFilterFragment(t *testing.T) {
	m := sized(testModel(), 120, 24)
	loadReleases(t, m, map[string][]charts.Release{
		"a": deployed("a", "c", "1.0.0"),
		"b": deployed("b", "c", "1.0.0"),
	}, nil)

	require.Contains(t, m.baseFooter(), "1/2")

	m.ApplySearchQuery("a")
	require.Contains(t, m.baseFooter(), "filtered from 2")
}

// The reason is the whole point of the health column: the column says something
// is wrong, the footer says what.
func TestFooterCarriesTheConvergenceReason(t *testing.T) {
	m := sized(testModel(), 120, 24)
	loadReleases(t, m,
		map[string][]charts.Release{"app": deployed("app", "c", "1.0.0")},
		map[string][]charts.ServiceState{"app": {{
			Name: "app_web", Running: 2, Desired: 3, NewestTaskAge: time.Hour,
		}}})

	footer := m.renderFooter()
	require.Contains(t, footer, "2/3 tasks running")
	require.Contains(t, footer, "app_web")
}

func TestFooterHasNoReasonWhenConverged(t *testing.T) {
	m := sized(testModel(), 120, 24)
	loadReleases(t, m,
		map[string][]charts.Release{"app": deployed("app", "c", "1.0.0")},
		map[string][]charts.ServiceState{"app": {converged("app_web")}},
		map[string]string{"c": "1.0.0"})

	require.Empty(t, m.selectedReason())
	footer := m.renderFooter()
	require.Equal(t, 2, strings.Count(footer, "\n")+1,
		"counts plus the read-only hint, and nothing else to report")
	require.Contains(t, footer, readOnlyHint)
}

// The boundary is stated before an operator presses a key and is told no.
func TestReadOnlyHintIsAlwaysOnScreen(t *testing.T) {
	m := sized(testModel(), 140, 24)
	require.Contains(t, m.View(), "Read-only", "even while loading")

	loadReleases(t, m, map[string][]charts.Release{"app": deployed("app", "c", "1.0.0")}, nil)
	require.Contains(t, m.View(), "Read-only")

	loadReleases(t, m, nil, nil)
	require.Contains(t, m.View(), "Read-only", "even with nothing installed")
}

// "nothing newer" and "nothing to compare against" are different answers, and
// one empty cell for both would let the second read as the first.
func TestUpdCellDistinguishesUnknownFromCurrent(t *testing.T) {
	m := sized(testModel(), 150, 24)

	loadReleases(t, m, map[string][]charts.Release{"app": deployed("app", "c", "1.0.0")}, nil)
	require.False(t, m.haveIndexes)
	require.Equal(t, "?", m.updCell(m.list.Filtered[0]), "no index cached")

	loadReleases(t, m,
		map[string][]charts.Release{"app": deployed("app", "c", "1.0.0")}, nil,
		map[string]string{"c": "1.0.0"})
	require.True(t, m.haveIndexes)
	require.Equal(t, "—", m.updCell(m.list.Filtered[0]), "index cached, nothing newer")

	loadReleases(t, m,
		map[string][]charts.Release{"app": deployed("app", "c", "1.0.0")}, nil,
		map[string]string{"c": "2.0.0"})
	require.Equal(t, "2.0.0", m.updCell(m.list.Filtered[0]))
}

// The footer carries the counts, the convergence reason and the read-only
// hint. A fourth line squeezes the content area enough to clip a dialog on a
// short terminal, so the missing-index signal lives in the column instead.
func TestFooterStaysWithinThreeLines(t *testing.T) {
	m := sized(testModel(), 140, 24)
	loadReleases(t, m,
		map[string][]charts.Release{"app": deployed("app", "c", "1.0.0")},
		map[string][]charts.ServiceState{"app": {{
			Name: "app_web", Running: 2, Desired: 3, NewestTaskAge: time.Hour,
		}}})

	require.NotEmpty(t, m.selectedReason(), "the fixture must produce a reason line")
	require.False(t, m.haveIndexes, "and have the missing-index condition too")
	require.LessOrEqual(t, strings.Count(m.renderFooter(), "\n")+1, 3)
}

func TestUpdColumnShowsTheNewerVersion(t *testing.T) {
	m := sized(testModel(), 150, 24)
	loadReleases(t, m,
		map[string][]charts.Release{
			"stale":   deployed("stale", "c", "1.0.0"),
			"current": deployed("current", "c2", "3.0.0"),
			"local":   deployed("local", "unpublished", "0.1.0"),
		}, nil,
		map[string]string{"c": "2.0.0", "c2": "3.0.0"})

	byName := map[string]releaseItem{}
	for _, it := range m.list.Filtered {
		byName[it.Name] = it
	}
	require.Equal(t, "2.0.0", byName["stale"].Latest)
	require.Empty(t, byName["current"].Latest, "already newest")
	require.Empty(t, byName["local"].Latest, "a chart in no index has nothing to be outdated against")

	out := m.View()
	require.Contains(t, out, "LATEST", "the header names the column")
	require.Contains(t, out, "2.0.0")
}
