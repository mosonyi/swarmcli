// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

//go:build integration

package charts

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"swarmcli/charts"
	swarmlog "swarmcli/utils/log"
)

// TestChartsApplyIsIdempotentAgainstARealSwarm is the one thing the unit tests
// cannot prove. Release history is one Docker Config per revision, so an apply
// that recorded a revision even when nothing changed would grow the swarm's
// config store on every CI run, forever. The fake backend can only simulate that;
// this counts the real Configs on a real swarm before and after a second apply.
func TestChartsApplyIsIdempotentAgainstARealSwarm(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()

	ctx := context.Background()
	release := fmt.Sprintf("itest-apply-%d", time.Now().UnixNano())

	chartDir := writeDemoChart(t)
	dir := t.TempDir()
	relFile := filepath.Join(dir, "swarmcli-release.yaml")
	require.NoError(t, os.WriteFile(relFile, []byte(fmt.Sprintf(
		"releases:\n  - name: %s\n    chart: %s\n", release, chartDir)), 0o600))

	rf, err := charts.LoadReleaseFile(relFile)
	require.NoError(t, err)

	eng := charts.NewEngine()
	src := charts.NewChartSource(nil) // local chart path: no repository needed
	defer func() { _, _ = eng.Uninstall(ctx, release, true) }()

	opts := charts.InstallOptions{Wait: true, Timeout: 90 * time.Second}

	// First apply installs.
	plan, err := eng.PlanApply(ctx, rf, src)
	require.NoError(t, err)
	require.Len(t, plan.Releases, 1)
	require.Equal(t, charts.ActionInstall, plan.Releases[0].Action)

	res, err := eng.Apply(ctx, plan, opts)
	require.NoError(t, err)
	require.Equal(t, 1, res[0].Revision)

	hist, err := eng.History(ctx, release)
	require.NoError(t, err)
	require.Len(t, hist, 1)

	// Second apply, nothing changed: it must plan `unchanged`, deploy nothing, and
	// record NO new revision.
	plan2, err := eng.PlanApply(ctx, rf, src)
	require.NoError(t, err)
	require.Equal(t, charts.ActionUnchanged, plan2.Releases[0].Action)

	res2, err := eng.Apply(ctx, plan2, opts)
	require.NoError(t, err)
	require.Equal(t, charts.ActionUnchanged, res2[0].Action)
	require.Zero(t, res2[0].Revision)

	hist2, err := eng.History(ctx, release)
	require.NoError(t, err)
	require.Len(t, hist2, 1, "a no-op apply must not record a revision on a real swarm")

	// A changed value is a real upgrade, so the mechanism is not simply inert.
	require.NoError(t, os.WriteFile(filepath.Join(dir, "v.yaml"), []byte("replicas: 2\n"), 0o600))
	require.NoError(t, os.WriteFile(relFile, []byte(fmt.Sprintf(
		"releases:\n  - name: %s\n    chart: %s\n    values: [./v.yaml]\n", release, chartDir)), 0o600))

	rf2, err := charts.LoadReleaseFile(relFile)
	require.NoError(t, err)
	plan3, err := eng.PlanApply(ctx, rf2, src)
	require.NoError(t, err)
	require.Equal(t, charts.ActionUpgrade, plan3.Releases[0].Action)

	res3, err := eng.Apply(ctx, plan3, opts)
	require.NoError(t, err)
	require.Equal(t, 2, res3[0].Revision)

	hist3, err := eng.History(ctx, release)
	require.NoError(t, err)
	require.Len(t, hist3, 2)
}

// apply never removes a release the file does not describe: a release records
// nothing about which manifest produced it, so a prune could not distinguish one
// owned by a second file, or installed by hand, from a genuinely obsolete one.
func TestChartsApplyLeavesUnmanagedReleasesRunning(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()

	ctx := context.Background()
	managed := fmt.Sprintf("itest-managed-%d", time.Now().UnixNano())
	byHand := fmt.Sprintf("itest-byhand-%d", time.Now().UnixNano())

	chartDir := writeDemoChart(t)
	eng := charts.NewEngine()
	src := charts.NewChartSource(nil)
	opts := charts.InstallOptions{Wait: true, Timeout: 90 * time.Second}

	defer func() {
		_, _ = eng.Uninstall(ctx, managed, true)
		_, _ = eng.Uninstall(ctx, byHand, true)
	}()

	// A release installed outside the file.
	ch, err := charts.LoadChartDir(chartDir)
	require.NoError(t, err)
	manifest, err := charts.Render(ch, charts.RenderContext{
		Values:  ch.Values,
		Release: charts.ReleaseMeta{Name: byHand, Namespace: byHand, Revision: 1},
		Chart:   charts.ChartMeta{Name: ch.Metadata.Name, Version: ch.Metadata.Version},
	})
	require.NoError(t, err)
	_, err = eng.Install(ctx, byHand, charts.ReleaseChartOf(ch), ch.Values, manifest, opts)
	require.NoError(t, err)

	dir := t.TempDir()
	relFile := filepath.Join(dir, "swarmcli-release.yaml")
	require.NoError(t, os.WriteFile(relFile, []byte(fmt.Sprintf(
		"releases:\n  - name: %s\n    chart: %s\n", managed, chartDir)), 0o600))
	rf, err := charts.LoadReleaseFile(relFile)
	require.NoError(t, err)

	plan, err := eng.PlanApply(ctx, rf, src)
	require.NoError(t, err)
	require.Contains(t, plan.Unmanaged, byHand, "the hand-installed release must be reported")

	_, err = eng.Apply(ctx, plan, opts)
	require.NoError(t, err)

	// It is still there.
	cur, _, err := eng.Status(ctx, byHand)
	require.NoError(t, err)
	require.Equal(t, charts.StatusDeployed, cur.Status, "apply must not touch an unmanaged release")
}
