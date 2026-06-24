// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
)

// seedRevisions installs a release and upgrades it to `count` total revisions
// using a fresh engine that does not auto-prune.
func seedRevisions(t *testing.T, e *Engine, release string, count int) {
	t.Helper()
	ctx := context.Background()
	_, err := e.Install(ctx, release, ReleaseChart{Name: "c", Version: "1"}, nil,
		"services:\n  s:\n    image: v1\n", InstallOptions{})
	require.NoError(t, err)
	for i := 2; i <= count; i++ {
		_, err := e.Upgrade(ctx, release, ReleaseChart{Name: "c", Version: "1"}, nil,
			fmt.Sprintf("services:\n  s:\n    image: v%d\n", i), InstallOptions{})
		require.NoError(t, err)
	}
}

func revs(n int) []Release {
	out := make([]Release, n)
	for i := range out {
		out[i] = Release{Revision: i + 1}
	}
	return out
}

func deleted(actions []PruneAction) []int {
	var out []int
	for _, a := range actions {
		if a.Delete {
			out = append(out, a.Revision)
		}
	}
	return out
}

func TestPlanPrune(t *testing.T) {
	cases := []struct {
		name string
		n    int
		keep int
		want []int // revisions deleted
	}{
		{"keep zero keeps all", 4, 0, nil},
		{"negative keeps all", 4, -3, nil},
		{"keep ge len keeps all", 4, 4, nil},
		{"keep gt len keeps all", 4, 9, nil},
		{"keep two deletes oldest two", 4, 2, []int{1, 2}},
		{"keep one deletes all but current", 4, 1, []int{1, 2, 3}},
		{"single revision never deleted", 1, 1, nil},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			actions := planPrune(revs(tc.n), tc.keep)
			require.Len(t, actions, tc.n)
			require.Equal(t, tc.want, deleted(actions))
			// The highest revision is always the current one and never deleted.
			top := actions[len(actions)-1]
			require.True(t, top.Current)
			require.False(t, top.Delete)
		})
	}
}

func TestPruneDeletesOldestKeepsCurrent(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	seedRevisions(t, e, "demo", 4)

	res, err := e.Prune(context.Background(), "demo", 2, false)
	require.NoError(t, err)
	require.Equal(t, []int{1, 2}, res.Deleted())

	_, ok1 := fb.configs[releaseConfigName("demo", 1)]
	_, ok2 := fb.configs[releaseConfigName("demo", 2)]
	_, ok3 := fb.configs[releaseConfigName("demo", 3)]
	_, ok4 := fb.configs[releaseConfigName("demo", 4)]
	require.False(t, ok1)
	require.False(t, ok2)
	require.True(t, ok3)
	require.True(t, ok4) // current revision retained
}

func TestPruneDryRunMutatesNothing(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	seedRevisions(t, e, "demo", 4)

	res, err := e.Prune(context.Background(), "demo", 2, true)
	require.NoError(t, err)
	require.Equal(t, []int{1, 2}, res.Deleted())
	require.Len(t, fb.configs, 4) // nothing actually deleted
}

func TestPruneKeepZeroIsNoOp(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	seedRevisions(t, e, "demo", 3)

	res, err := e.Prune(context.Background(), "demo", 0, false)
	require.NoError(t, err)
	require.Empty(t, res.Deleted())
	require.Len(t, fb.configs, 3)
}

func TestPruneNotFound(t *testing.T) {
	e := testEngine(newFakeBackend())
	_, err := e.Prune(context.Background(), "ghost", 1, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not found")
}

func TestPruneAllPerReleaseIsolation(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	seedRevisions(t, e, "alpha", 3)
	seedRevisions(t, e, "beta", 2)

	results, err := e.PruneAll(context.Background(), 1, false)
	require.NoError(t, err)
	require.Len(t, results, 2)
	require.Equal(t, "alpha", results[0].Release) // sorted by name
	require.Equal(t, "beta", results[1].Release)

	// Each release keeps only its current revision.
	require.Equal(t, []int{1, 2}, results[0].Deleted())
	require.Equal(t, []int{1}, results[1].Deleted())
	require.Len(t, fb.configs, 2)
	require.Contains(t, fb.configs, releaseConfigName("alpha", 3))
	require.Contains(t, fb.configs, releaseConfigName("beta", 2))
}

func TestPruneAggregatesDeleteErrors(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	seedRevisions(t, e, "demo", 3)
	fb.deleteCfgErr = map[string]error{releaseConfigName("demo", 1): fmt.Errorf("boom")}

	_, err := e.Prune(context.Background(), "demo", 1, false)
	require.Error(t, err)
	require.Contains(t, err.Error(), "boom")
	// The non-failing revision was still deleted; the failing one remains.
	require.Contains(t, fb.configs, releaseConfigName("demo", 1))
	require.NotContains(t, fb.configs, releaseConfigName("demo", 2))
	require.Contains(t, fb.configs, releaseConfigName("demo", 3)) // current
}
