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

	"github.com/Eldara-Tech/swarmcli/charts"
	"github.com/Eldara-Tech/swarmcli/docker"
	swarmlog "github.com/Eldara-Tech/swarmcli/utils/log"
)

// writeOwnedReleaseFile writes a release file for one local chart, with the
// given owner (empty for none), and loads it.
func writeOwnedReleaseFile(t *testing.T, dir, release, chartDir, owner string) *charts.ReleaseFile {
	t.Helper()
	body := ""
	if owner != "" {
		body += "owner: " + owner + "\n"
	}
	body += fmt.Sprintf("releases:\n  - name: %s\n    chart: %s\n", release, chartDir)
	path := filepath.Join(dir, "swarmcli-release.yaml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	rf, err := charts.LoadReleaseFile(path)
	require.NoError(t, err)
	return rf
}

// ownerLabelOf reads the com.swarmcli.owner label off a revision's real Docker
// Config. The stored record carries the owner too, but the label is what a
// prune sweep actually classifies from, so that is what has to be right.
func ownerLabelOf(t *testing.T, ctx context.Context, release string, rev int) (string, bool) {
	t.Helper()
	want := fmt.Sprintf("swarmcli.release.%s.v%d", release, rev)
	cfgs, err := docker.ListConfigs(ctx)
	require.NoError(t, err)
	for _, c := range cfgs {
		if c.Spec.Name == want {
			v, ok := c.Spec.Labels[charts.LabelOwner]
			return v, ok
		}
	}
	t.Fatalf("no Docker Config %q on the swarm", want)
	return "", false
}

// Taking a release over with an otherwise identical file must move the stamp,
// on a real swarm and on the real Docker Config label.
//
// The unit tests prove the plan says "upgrade". They cannot prove the label
// round-trips: the stamp is written as a Config label by deployAndRecord and
// read back by List, and a prune sweep in swarmcli-cd classifies from that
// label. Reproduced there as a would-be destructive delete of a running stack
// (#511, swarmcli-cd#62).
func TestChartsApplyCorrectsAStaleOwnerStampOnARealSwarm(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()

	ctx := context.Background()
	release := fmt.Sprintf("itest-owner-%d", time.Now().UnixNano())
	chartDir := writeDemoChart(t)
	dir := t.TempDir()

	eng := charts.NewEngine()
	src := charts.NewChartSource(nil)
	defer func() { _, _ = eng.Uninstall(ctx, release, true) }()

	opts := charts.InstallOptions{Wait: true, Timeout: 90 * time.Second}

	// Installed under the first owner.
	rf := writeOwnedReleaseFile(t, dir, release, chartDir, "old-app")
	plan, err := eng.PlanApply(ctx, rf, src, charts.PlanOptions{})
	require.NoError(t, err)
	require.Equal(t, charts.ActionInstall, plan.Releases[0].Action)
	res, err := eng.Apply(ctx, plan, charts.InstallOptions{
		Wait: opts.Wait, Timeout: opts.Timeout, Owner: plan.Owner})
	require.NoError(t, err)
	require.Equal(t, 1, res[0].Revision)

	got, ok := ownerLabelOf(t, ctx, release, 1)
	require.True(t, ok, "revision 1 must carry an owner label")
	require.Equal(t, "apply/old-app:release/"+release, got)

	// Only the owner changes. Chart, values and rendered manifest are identical,
	// so before the fix this planned as unchanged and the stamp never moved.
	rf2 := writeOwnedReleaseFile(t, dir, release, chartDir, "new-app")
	plan2, err := eng.PlanApply(ctx, rf2, src, charts.PlanOptions{})
	require.NoError(t, err)
	require.Equal(t, charts.ActionUpgrade, plan2.Releases[0].Action,
		"a handover is a real change even when the content is byte-identical")

	res2, err := eng.Apply(ctx, plan2, charts.InstallOptions{
		Wait: opts.Wait, Timeout: opts.Timeout, Owner: plan2.Owner})
	require.NoError(t, err)
	require.Equal(t, 2, res2[0].Revision)

	got, ok = ownerLabelOf(t, ctx, release, 2)
	require.True(t, ok)
	require.Equal(t, "apply/new-app:release/"+release, got,
		"the new revision is stamped with whoever wrote it")

	// And it settles: the new owner's next apply is a no-op. A handover that
	// re-applied for ever would be worse than the bug it fixes.
	hist, err := eng.History(ctx, release)
	require.NoError(t, err)
	require.Len(t, hist, 2)

	plan3, err := eng.PlanApply(ctx, rf2, src, charts.PlanOptions{})
	require.NoError(t, err)
	require.Equal(t, charts.ActionUnchanged, plan3.Releases[0].Action)
	res3, err := eng.Apply(ctx, plan3, charts.InstallOptions{
		Wait: opts.Wait, Timeout: opts.Timeout, Owner: plan3.Owner})
	require.NoError(t, err)
	require.Zero(t, res3[0].Revision)

	hist3, err := eng.History(ctx, release)
	require.NoError(t, err)
	require.Len(t, hist3, 2, "a settled handover must not keep writing revisions")
}

// THE MIGRATION GUARANTEE, on a real swarm.
//
// A release installed before owner stamping existed carries no stamp. Planning
// it against an owner must stay ActionUnchanged. If it does not, the first
// `charts apply` after upgrading re-deploys every release on the swarm — every
// spec re-pushed, and with --resolve-image always, every digest re-resolved —
// to buy safety it already had, because an unstamped release is classified
// Unmanaged rather than Orphaned and so was never a prune candidate.
//
// The unit test asserts the plan. This asserts the swarm: no new Config, and
// the running service untouched. Do not delete this test.
func TestChartsApplyDoesNotChurnAnUnstampedReleaseOnARealSwarm(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()

	ctx := context.Background()
	release := fmt.Sprintf("itest-nochurn-%d", time.Now().UnixNano())
	chartDir := writeDemoChart(t)
	dir := t.TempDir()

	eng := charts.NewEngine()
	src := charts.NewChartSource(nil)
	defer func() { _, _ = eng.Uninstall(ctx, release, true) }()

	opts := charts.InstallOptions{Wait: true, Timeout: 90 * time.Second}

	// Installed with no owner at all — what every release deployed before owner
	// stamping shipped looks like.
	rf := writeOwnedReleaseFile(t, dir, release, chartDir, "")
	plan, err := eng.PlanApply(ctx, rf, src, charts.PlanOptions{})
	require.NoError(t, err)
	require.Empty(t, plan.Owner)
	_, err = eng.Apply(ctx, plan, opts)
	require.NoError(t, err)

	_, ok := ownerLabelOf(t, ctx, release, 1)
	require.False(t, ok, "the fixture has to be unstamped for this to test anything")

	// Now an apply that DOES claim an owner. The content is identical.
	rf2 := writeOwnedReleaseFile(t, dir, release, chartDir, "prod")
	plan2, err := eng.PlanApply(ctx, rf2, src, charts.PlanOptions{})
	require.NoError(t, err)
	require.Equal(t, charts.ActionUnchanged, plan2.Releases[0].Action,
		"an unstamped release must not be re-deployed just to acquire a stamp")

	res2, err := eng.Apply(ctx, plan2, charts.InstallOptions{
		Wait: opts.Wait, Timeout: opts.Timeout, Owner: plan2.Owner})
	require.NoError(t, err)
	require.Zero(t, res2[0].Revision)

	hist, err := eng.History(ctx, release)
	require.NoError(t, err)
	require.Len(t, hist, 1, "no revision may be written on a real swarm")
}

// An apply that claims nothing must not strip a stamp somebody else wrote.
func TestChartsApplyWithoutAnOwnerLeavesAnExistingStamp(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()

	ctx := context.Background()
	release := fmt.Sprintf("itest-noclaim-%d", time.Now().UnixNano())
	chartDir := writeDemoChart(t)
	dir := t.TempDir()

	eng := charts.NewEngine()
	src := charts.NewChartSource(nil)
	defer func() { _, _ = eng.Uninstall(ctx, release, true) }()

	opts := charts.InstallOptions{Wait: true, Timeout: 90 * time.Second}

	rf := writeOwnedReleaseFile(t, dir, release, chartDir, "someone-else")
	plan, err := eng.PlanApply(ctx, rf, src, charts.PlanOptions{})
	require.NoError(t, err)
	_, err = eng.Apply(ctx, plan, charts.InstallOptions{
		Wait: opts.Wait, Timeout: opts.Timeout, Owner: plan.Owner})
	require.NoError(t, err)

	rf2 := writeOwnedReleaseFile(t, dir, release, chartDir, "")
	plan2, err := eng.PlanApply(ctx, rf2, src, charts.PlanOptions{})
	require.NoError(t, err)
	require.Equal(t, charts.ActionUnchanged, plan2.Releases[0].Action)
	_, err = eng.Apply(ctx, plan2, opts)
	require.NoError(t, err)

	got, ok := ownerLabelOf(t, ctx, release, 1)
	require.True(t, ok)
	require.Equal(t, "apply/someone-else:release/"+release, got,
		"an unowned apply must not strip another owner's stamp")
}
