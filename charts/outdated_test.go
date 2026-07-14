// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func idx(chart string, versions ...string) *Index {
	entries := make([]IndexEntry, 0, len(versions))
	for _, v := range versions {
		entries = append(entries, IndexEntry{Name: chart, Version: v})
	}
	return &Index{APIVersion: "v1", Entries: map[string][]IndexEntry{chart: entries}}
}

func rel(name, chart, version string) Release {
	return Release{Name: name, Chart: ReleaseChart{Name: chart, Version: version}}
}

func TestOutdated(t *testing.T) {
	got := Outdated(
		[]Release{rel("hello", "whoami", "0.1.6")},
		map[string]*Index{"swarmcli-charts": idx("whoami", "0.1.6", "0.1.8", "0.1.7")},
	)
	require.Len(t, got, 1)
	require.Equal(t, OutdatedEntry{
		Release: "hello", Chart: "whoami", Repo: "swarmcli-charts",
		Installed: "0.1.6", Latest: "0.1.8",
	}, got[0])
}

func TestOutdatedSkipsCurrentReleases(t *testing.T) {
	got := Outdated(
		[]Release{rel("hello", "whoami", "0.1.8")},
		map[string]*Index{"swarmcli-charts": idx("whoami", "0.1.8")},
	)
	require.Empty(t, got)
}

// A local chart is in no index; that is not an error, it is just not reportable.
func TestOutdatedSkipsChartsInNoIndex(t *testing.T) {
	got := Outdated(
		[]Release{rel("hello", "mine", "0.1.0")},
		map[string]*Index{"swarmcli-charts": idx("whoami", "0.1.8")},
	)
	require.Empty(t, got)
}

// Someone running a version newer than any index offers is not "outdated".
func TestOutdatedIgnoresReleasesAheadOfTheIndex(t *testing.T) {
	got := Outdated(
		[]Release{rel("hello", "whoami", "0.2.0")},
		map[string]*Index{"swarmcli-charts": idx("whoami", "0.1.8")},
	)
	require.Empty(t, got)
}

// A Release does not record which repository it came from, so the highest version
// across all of them wins and the reported repo is the one that supplied it.
func TestOutdatedPicksHighestAcrossRepos(t *testing.T) {
	got := Outdated(
		[]Release{rel("hello", "whoami", "0.1.0")},
		map[string]*Index{
			"stable": idx("whoami", "0.1.2"),
			"edge":   idx("whoami", "0.2.0"),
		},
	)
	require.Len(t, got, 1)
	require.Equal(t, "0.2.0", got[0].Latest)
	require.Equal(t, "edge", got[0].Repo)
}

func TestOutdatedSortsByRelease(t *testing.T) {
	got := Outdated(
		[]Release{rel("zeta", "whoami", "0.1.0"), rel("alpha", "whoami", "0.1.0")},
		map[string]*Index{"r": idx("whoami", "0.1.8")},
	)
	require.Len(t, got, 2)
	require.Equal(t, "alpha", got[0].Release)
	require.Equal(t, "zeta", got[1].Release)
}
