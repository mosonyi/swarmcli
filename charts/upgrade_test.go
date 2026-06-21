// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func mustInstall(t *testing.T, e *Engine, rel, manifest string) {
	t.Helper()
	_, err := e.Install(context.Background(), rel, ReleaseChart{Name: "c", Version: "1"}, map[string]any{"v": 1}, manifest, InstallOptions{})
	require.NoError(t, err)
}

func TestUpgradeAddsRevisionAndSupersedes(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	ctx := context.Background()
	mustInstall(t, e, "demo", "services:\n  s:\n    image: a\n")

	rel, err := e.Upgrade(ctx, "demo", ReleaseChart{Name: "c", Version: "2"}, map[string]any{"v": 2}, "services:\n  s:\n    image: b\n", InstallOptions{})
	require.NoError(t, err)
	require.Equal(t, 2, rel.Revision)
	require.Equal(t, "services:\n  s:\n    image: b\n", fb.deployed["demo"])

	hist, err := e.History(ctx, "demo")
	require.NoError(t, err)
	require.Len(t, hist, 2)
	require.Equal(t, StatusSuperseded, hist[0].Status) // rev1 derived superseded
	require.Equal(t, StatusDeployed, hist[1].Status)
}

func TestUpgradeMissingReleaseRequiresInstallFlag(t *testing.T) {
	e := testEngine(newFakeBackend())
	ctx := context.Background()
	_, err := e.Upgrade(ctx, "nope", ReleaseChart{Name: "c", Version: "1"}, nil, "services:\n  s:\n    image: a\n", InstallOptions{})
	require.Error(t, err)

	rel, err := e.Upgrade(ctx, "nope", ReleaseChart{Name: "c", Version: "1"}, nil, "services:\n  s:\n    image: a\n", InstallOptions{Install: true})
	require.NoError(t, err)
	require.Equal(t, 1, rel.Revision)
}

func TestRollbackCopiesTargetRevision(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	ctx := context.Background()
	mustInstall(t, e, "demo", "services:\n  s:\n    image: v1\n")
	_, err := e.Upgrade(ctx, "demo", ReleaseChart{Name: "c", Version: "2"}, map[string]any{"v": 2}, "services:\n  s:\n    image: v2\n", InstallOptions{})
	require.NoError(t, err)

	rb, err := e.Rollback(ctx, "demo", 1, InstallOptions{})
	require.NoError(t, err)
	require.Equal(t, 3, rb.Revision)                                  // append-only
	require.Equal(t, "services:\n  s:\n    image: v1\n", rb.Manifest) // content from rev1
	require.Equal(t, "services:\n  s:\n    image: v1\n", fb.deployed["demo"])

	cur, _, err := e.Status(ctx, "demo")
	require.NoError(t, err)
	require.Equal(t, 3, cur.Revision)
}

func TestRollbackUnknownRevision(t *testing.T) {
	e := testEngine(newFakeBackend())
	ctx := context.Background()
	mustInstall(t, e, "demo", "services:\n  s:\n    image: v1\n")
	_, err := e.Rollback(ctx, "demo", 9, InstallOptions{})
	require.Error(t, err)
}

func TestGetRevision(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	ctx := context.Background()
	mustInstall(t, e, "demo", "services:\n  s:\n    image: v1\n")
	_, err := e.Upgrade(ctx, "demo", ReleaseChart{Name: "c", Version: "2"}, map[string]any{"v": 2}, "services:\n  s:\n    image: v2\n", InstallOptions{})
	require.NoError(t, err)

	cur, err := e.GetRevision(ctx, "demo", 0)
	require.NoError(t, err)
	require.Equal(t, 2, cur.Revision)

	r1, err := e.GetRevision(ctx, "demo", 1)
	require.NoError(t, err)
	require.Contains(t, r1.Manifest, "v1")
}

func TestHistoryMaxPrunes(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	ctx := context.Background()
	_, err := e.Install(ctx, "demo", ReleaseChart{Name: "c", Version: "1"}, nil, "services:\n  s:\n    image: v1\n", InstallOptions{HistoryMax: 2})
	require.NoError(t, err)
	for i := 2; i <= 4; i++ {
		_, err := e.Upgrade(ctx, "demo", ReleaseChart{Name: "c", Version: "x"}, nil, "services:\n  s:\n    image: v\n", InstallOptions{HistoryMax: 2})
		require.NoError(t, err)
	}
	hist, err := e.History(ctx, "demo")
	require.NoError(t, err)
	require.Len(t, hist, 2) // only the 2 most recent retained
	require.Equal(t, 3, hist[0].Revision)
	require.Equal(t, 4, hist[1].Revision)
}
