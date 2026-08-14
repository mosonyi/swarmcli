// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

//go:build integration

package charts

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Eldara-Tech/swarmcli/charts"
	swarmlog "github.com/Eldara-Tech/swarmcli/utils/log"
)

// renderRevision renders the demo chart at a replica count, as an install or
// upgrade would.
func renderRevision(t *testing.T, ch *charts.Chart, release string, rev int, replicas string) (string, map[string]any) {
	t.Helper()
	values, err := charts.MergeValues(ch.Values, nil, []string{"replicas=" + replicas})
	require.NoError(t, err)
	manifest, err := charts.Render(ch, charts.RenderContext{
		Values:  values,
		Release: charts.ReleaseMeta{Name: release, Namespace: release, Revision: rev},
		Chart:   charts.ChartMeta{Name: ch.Metadata.Name, Version: ch.Metadata.Version},
	})
	require.NoError(t, err)
	return manifest, values
}

// TestChartsBrowserReadPath exercises what the :charts view renders, against a
// real swarm: AllRevisions for the rows and the expansion, and Rollup over the
// live service states for the health column.
//
// The view itself is covered by unit tests that need no Docker. What cannot be
// faked is whether these two engine reads agree with a swarm that really has a
// release on it — a rollout that reads converged here is the assertion the
// health column ultimately rests on.
func TestChartsBrowserReadPath(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()

	ctx := context.Background()
	release := fmt.Sprintf("itest-browser-%d", time.Now().UnixNano())

	ch, err := charts.LoadChartDir(writeDemoChart(t))
	require.NoError(t, err)
	rc := charts.ReleaseChart{Name: ch.Metadata.Name, Version: ch.Metadata.Version}

	eng := charts.NewEngine()
	defer func() { _, _ = eng.Uninstall(ctx, release, true) }()

	manifest, values := renderRevision(t, ch, release, 1, "1")
	_, err = eng.Install(ctx, release, rc, values, manifest,
		charts.InstallOptions{Wait: true, Timeout: 90 * time.Second})
	require.NoError(t, err)

	upManifest, upValues := renderRevision(t, ch, release, 2, "2")
	_, err = eng.Upgrade(ctx, release, rc, upValues, upManifest,
		charts.InstallOptions{Wait: true, Timeout: 90 * time.Second})
	require.NoError(t, err)

	// One read gives the list rows and every expanded release's history.
	all, err := eng.AllRevisions(ctx)
	require.NoError(t, err)
	revs := all[release]
	require.Len(t, revs, 2, "the expansion shows both revisions")
	require.Equal(t, 1, revs[0].Revision, "ascending, as the expansion renders them")
	require.Equal(t, 2, revs[1].Revision)
	require.Equal(t, charts.StatusSuperseded, revs[0].Status,
		"the older revision reads superseded in the expansion")
	require.Equal(t, charts.StatusDeployed, revs[1].Status)

	// Both manifests are stored, which is what makes the in-view diff possible
	// without a chart, a re-render or a network.
	require.NotEqual(t, revs[0].Manifest, revs[1].Manifest)
	require.Contains(t, revs[0].Manifest, "replicas: 1")
	require.Contains(t, revs[1].Manifest, "replicas: 2")

	// The health column: a rollout that finished must read converged, since
	// both deploys above ran with --wait.
	states := eng.Backend.StackServices(ctx, release)
	require.NotEmpty(t, states, "the expansion lists the running services")
	conv := charts.Rollup(states)
	require.Equal(t, charts.PhaseConverged, conv.Phase,
		"a waited rollout must read converged: %s", conv.Reason)
	require.Empty(t, conv.Reason, "a converged release has nothing to explain")
}
