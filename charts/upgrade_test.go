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

// TestRollbackReplaysTheFilesTheOriginalDeploySent is the end-to-end test for
// the file chain, and the only one that crosses all six of its links:
// InstallOptions.Files -> Release.Files -> storeRevision's gzip -> decodeRelease
// -> newRevision on the rollback path -> DeployRequest.Files.
//
// It has to be end to end because every link fails the same silent way. A drop
// anywhere yields an empty map, and an empty map is exactly what a chart with no
// files/ directory legitimately produces — so nothing downstream can tell the
// two apart, and a test of any single link passes while the chain is severed.
//
// Rollback is where it matters: it holds no chart, no ChartSource and no
// filesystem, so unless the bytes came back out of the stored record there is
// nowhere else they could have come from.
func TestRollbackReplaysTheFilesTheOriginalDeploySent(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	ctx := context.Background()
	const manifest = "services:\n  web:\n    image: nginx\nconfigs:\n  site:\n    file: files/nginx.conf\n"

	v1 := map[string][]byte{"files/nginx.conf": []byte("server { listen 80; }")}
	v2 := map[string][]byte{"files/nginx.conf": []byte("server { listen 8080; }")}

	_, err := e.Install(ctx, "demo", ReleaseChart{Name: "c", Version: "1"}, nil, manifest, InstallOptions{Files: v1})
	require.NoError(t, err)
	require.Equal(t, v1, fb.lastDeploy.Files)

	// The manifest is unchanged between the revisions, so only the files can
	// distinguish what rev 3 deploys from what rev 2 did.
	_, err = e.Upgrade(ctx, "demo", ReleaseChart{Name: "c", Version: "2"}, nil, manifest, InstallOptions{Files: v2})
	require.NoError(t, err)
	require.Equal(t, v2, fb.lastDeploy.Files)

	// No Files in the options: whatever rev 3 deploys, it read off rev 1.
	rb, err := e.Rollback(ctx, "demo", 1, InstallOptions{})
	require.NoError(t, err)
	require.Equal(t, 3, rb.Revision)
	require.Equal(t, v1, fb.lastDeploy.Files, "a rollback must deploy the bytes the revision it replays deployed")
	require.Equal(t, v1, map[string][]byte(rb.Files), "and record them, so rolling back to the rollback replays them too")

	// Through the whole store/decode round trip once more, from the record
	// rather than from the value Rollback returned.
	stored, err := e.GetRevision(ctx, "demo", 3)
	require.NoError(t, err)
	require.Equal(t, v1, map[string][]byte(stored.Files))
}

// A chart with no files/ directory must keep behaving exactly as it did before
// any of this existed: nothing recorded, nothing deployed, and nil rather than
// an empty map at every step — the round trip through YAML and gzip is where a
// nil would otherwise come back as something else.
func TestRollbackOfAReleaseWithoutFilesCarriesNone(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	ctx := context.Background()
	mustInstall(t, e, "demo", "services:\n  s:\n    image: v1\n")
	_, err := e.Upgrade(ctx, "demo", ReleaseChart{Name: "c", Version: "2"}, nil, "services:\n  s:\n    image: v2\n", InstallOptions{})
	require.NoError(t, err)

	rb, err := e.Rollback(ctx, "demo", 1, InstallOptions{})
	require.NoError(t, err)
	require.Nil(t, rb.Files)
	require.Nil(t, fb.lastDeploy.Files)

	stored, err := e.GetRevision(ctx, "demo", 1)
	require.NoError(t, err)
	require.Nil(t, stored.Files)
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
