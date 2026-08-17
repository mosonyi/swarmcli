// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/Eldara-Tech/swarmcli/charts"

	"github.com/stretchr/testify/require"
)

// seedRepoCache points the charts state directory at a temp dir holding one
// configured repository and its cached index, exactly as `charts repo add`
// would have left it.
func seedRepoCache(t *testing.T, chart string, versions ...string) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", dir)

	base := filepath.Join(dir, "swarmcli", "charts")
	require.NoError(t, os.MkdirAll(filepath.Join(base, "cache"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(base, "repos.json"),
		[]byte(`[{"name":"testrepo","url":"https://charts.example.com"}]`), 0o600))

	index := "apiVersion: v1\nentries:\n  " + chart + ":\n"
	for _, v := range versions {
		index += "    - name: " + chart + "\n      version: \"" + v + "\"\n"
	}
	require.NoError(t, os.WriteFile(
		filepath.Join(base, "cache", "index-testrepo.yaml"), []byte(index), 0o600))
	return base
}

// The view polls every few seconds, and RepoStore's methods take no context, so
// a fetch started here could not even be cancelled when the view closes. The
// policy is what keeps that from happening, so assert the policy.
func TestRepoStoreNeverGoesToTheNetwork(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	store, err := newRepoStore()
	require.NoError(t, err)
	require.Equal(t, charts.RefreshNever, store.Refresh,
		"any other policy lets the poll fetch an index on its own initiative")
	require.NotNil(t, store.Warnf, "charts is silent unless the embedder wires this")
}

func TestOutdatedReadsTheCachedIndex(t *testing.T) {
	seedRepoCache(t, "mychart", "1.0.0", "2.1.0")

	ops := &engineOps{}
	entries, haveIndexes := ops.Outdated([]charts.Release{{
		Name: "app", Chart: charts.ReleaseChart{Name: "mychart", Version: "1.0.0"},
	}})

	require.True(t, haveIndexes)
	require.Len(t, entries, 1)
	require.Equal(t, "app", entries[0].Release)
	require.Equal(t, "1.0.0", entries[0].Installed)
	require.Equal(t, "2.1.0", entries[0].Latest)
}

func TestOutdatedReportsNoIndexesWhenNoneAreCached(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	ops := &engineOps{}
	entries, haveIndexes := ops.Outdated([]charts.Release{{
		Name: "app", Chart: charts.ReleaseChart{Name: "mychart", Version: "1.0.0"},
	}})

	require.False(t, haveIndexes, "no cached index is not the same answer as up to date")
	require.Empty(t, entries)
}

// The indexes are read once per view, which is what lets the poll ask on every
// tick without re-parsing them.
func TestIndexesAreReadOnce(t *testing.T) {
	base := seedRepoCache(t, "mychart", "1.0.0", "2.1.0")
	ops := &engineOps{}

	rels := []charts.Release{{Name: "app", Chart: charts.ReleaseChart{Name: "mychart", Version: "1.0.0"}}}
	first, haveIndexes := ops.Outdated(rels)
	require.True(t, haveIndexes)
	require.Len(t, first, 1)

	// Remove the cache entirely: a second read from disk would now find
	// nothing, so an equal answer can only have come from what was already
	// loaded.
	require.NoError(t, os.RemoveAll(base))

	second, haveIndexes := ops.Outdated(rels)
	require.True(t, haveIndexes)
	require.Equal(t, first, second)
}
