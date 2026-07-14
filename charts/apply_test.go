// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
)

// fakeChartSource serves charts from testdata without a repository, a network or
// a filesystem lookup, which is the whole point of the ChartSource seam.
type fakeChartSource struct {
	charts map[string]*Chart // "<repo>/<chart>@<version>" -> chart
	loads  int
}

func (f *fakeChartSource) Load(ref, version string) (*Chart, error) {
	f.loads++
	if ch, ok := f.charts[ref+"@"+version]; ok {
		return ch, nil
	}
	return nil, fmt.Errorf("chart %q version %q not found", ref, version)
}

// demoChart loads testdata/demo and stamps it with a version, so a plan can move
// a release from one chart version to another.
func demoChart(t *testing.T, version string) *Chart {
	t.Helper()
	ch, err := LoadChartDir("testdata/demo")
	require.NoError(t, err)
	ch.Metadata.Version = version
	return ch
}

// applyEnv wires an Engine over the fake backend with a fake chart source and
// writes the release file to a temp dir.
func applyEnv(t *testing.T, body string, versions ...string) (*Engine, *fakeBackend, *fakeChartSource, *ReleaseFile) {
	t.Helper()
	fb := newFakeBackend()
	e := NewEngineWith(fb)

	src := &fakeChartSource{charts: map[string]*Chart{}}
	for _, v := range versions {
		src.charts["swarmcli-charts/demo@"+v] = demoChart(t, v)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "swarmcli-release.yaml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	rf, err := LoadReleaseFile(path)
	require.NoError(t, err)
	return e, fb, src, rf
}

const oneRelease = `repositories:
  - name: swarmcli-charts
    url: https://eldara-tech.github.io/swarmcli-charts
releases:
  - name: hello
    chart: swarmcli-charts/demo
    version: "0.1.0"
`

func TestApplyInstallsThenIsIdempotent(t *testing.T) {
	e, fb, src, rf := applyEnv(t, oneRelease, "0.1.0")
	ctx := context.Background()

	plan, err := e.PlanApply(ctx, rf, src)
	require.NoError(t, err)
	require.Len(t, plan.Releases, 1)
	require.Equal(t, ActionInstall, plan.Releases[0].Action)

	res, err := e.Apply(ctx, plan, InstallOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, res[0].Revision)
	require.Contains(t, fb.deployed, "hello")

	// THE NO-CHURN GUARANTEE. History is one Docker Config per revision, so an
	// apply that recorded a revision when nothing changed would grow the swarm's
	// config store on every CI run, forever. Re-applying an unchanged file must
	// write nothing at all. Do not delete this test.
	before := len(fb.configs)

	plan2, err := e.PlanApply(ctx, rf, src)
	require.NoError(t, err)
	require.Equal(t, ActionUnchanged, plan2.Releases[0].Action)

	res2, err := e.Apply(ctx, plan2, InstallOptions{})
	require.NoError(t, err)
	require.Equal(t, ActionUnchanged, res2[0].Action)
	require.Zero(t, res2[0].Revision)
	require.Len(t, fb.configs, before, "an unchanged apply must not record a revision")
}

func TestApplyUpgradesOnVersionBump(t *testing.T) {
	e, _, src, rf := applyEnv(t, oneRelease, "0.1.0")
	ctx := context.Background()

	plan, err := e.PlanApply(ctx, rf, src)
	require.NoError(t, err)
	_, err = e.Apply(ctx, plan, InstallOptions{})
	require.NoError(t, err)

	// This is exactly what Renovate does to the file: bump the pinned version.
	bumped := &ReleaseFile{
		Path: rf.Path, Dir: rf.Dir,
		Releases: []ReleaseSpec{{Name: "hello", Chart: "swarmcli-charts/demo", Version: "0.2.0"}},
	}
	src.charts["swarmcli-charts/demo@0.2.0"] = demoChart(t, "0.2.0")

	plan2, err := e.PlanApply(ctx, bumped, src)
	require.NoError(t, err)
	require.Equal(t, ActionUpgrade, plan2.Releases[0].Action)
	require.Equal(t, "0.1.0", plan2.Releases[0].FromVersion)
	require.Equal(t, "0.2.0", plan2.Releases[0].ToVersion)

	res, err := e.Apply(ctx, plan2, InstallOptions{})
	require.NoError(t, err)
	require.Equal(t, 2, res[0].Revision)

	cur, err := e.GetRevision(ctx, "hello", 0)
	require.NoError(t, err)
	require.Equal(t, "0.2.0", cur.Chart.Version)
}

// A values change that the template does not read still changes the release's
// recorded values, so it must still be an upgrade. This proves the values half of
// the comparison is live rather than decorative.
func TestApplyDetectsValuesOnlyChange(t *testing.T) {
	e, _, src, rf := applyEnv(t, oneRelease, "0.1.0")
	ctx := context.Background()

	plan, err := e.PlanApply(ctx, rf, src)
	require.NoError(t, err)
	_, err = e.Apply(ctx, plan, InstallOptions{})
	require.NoError(t, err)

	vals := filepath.Join(rf.Dir, "extra.yaml")
	require.NoError(t, os.WriteFile(vals, []byte("unreadByTemplate: 1\n"), 0o600))
	withValues := &ReleaseFile{
		Path: rf.Path, Dir: rf.Dir,
		Releases: []ReleaseSpec{{
			Name: "hello", Chart: "swarmcli-charts/demo", Version: "0.1.0",
			Values: []string{"./extra.yaml"},
		}},
	}

	plan2, err := e.PlanApply(ctx, withValues, src)
	require.NoError(t, err)
	require.Equal(t, ActionUpgrade, plan2.Releases[0].Action)
}

// The stored values come back through YAML, so a value merged in memory as
// float64(1.0) may be decoded as int(1). reflect.DeepEqual would call those
// different and every apply would churn a revision. Canonicalising both through
// the same encoder must make them compare equal.
func TestSameValuesIgnoresYAMLTypeSkew(t *testing.T) {
	stored := map[string]any{"replicas": int(1), "ratio": float64(0.5)}
	merged := map[string]any{"replicas": float64(1.0), "ratio": float64(0.5)}

	same, err := sameValues(stored, merged)
	require.NoError(t, err)
	require.True(t, same, "int(1) and float64(1.0) must not look like a change")

	diff, err := sameValues(stored, map[string]any{"replicas": 2})
	require.NoError(t, err)
	require.False(t, diff)
}

// Key order must not matter either.
func TestSameValuesIsOrderIndependent(t *testing.T) {
	same, err := sameValues(
		map[string]any{"a": 1, "b": map[string]any{"x": 1, "y": 2}},
		map[string]any{"b": map[string]any{"y": 2, "x": 1}, "a": 1},
	)
	require.NoError(t, err)
	require.True(t, same)
}

// apply never deletes. A release on the swarm that the file does not mention is
// reported and left running.
func TestApplyNeverRemovesUnmanagedReleases(t *testing.T) {
	e, fb, src, rf := applyEnv(t, oneRelease, "0.1.0")
	ctx := context.Background()

	// A release installed by hand, or by a second manifest.
	ch := demoChart(t, "0.1.0")
	_, err := e.Install(ctx, "legacy", ReleaseChartOf(ch), ch.Values, "services: {}\n", InstallOptions{})
	require.NoError(t, err)

	plan, err := e.PlanApply(ctx, rf, src)
	require.NoError(t, err)
	require.Equal(t, []string{"legacy"}, plan.Unmanaged)

	_, err = e.Apply(ctx, plan, InstallOptions{})
	require.NoError(t, err)
	require.Contains(t, fb.deployed, "legacy", "apply must not remove a release it does not manage")
	require.Contains(t, fb.deployed, "hello")
}

// Everything is rendered and validated before anything is deployed, so a bad
// release cannot leave the swarm half-converged.
func TestPlanApplyValidatesEverythingBeforeDeployingAnything(t *testing.T) {
	body := `releases:
  - name: good
    chart: swarmcli-charts/demo
    version: "0.1.0"
  - name: bad
    chart: swarmcli-charts/demo
    version: "9.9.9"
`
	e, fb, src, rf := applyEnv(t, body, "0.1.0")

	_, err := e.PlanApply(context.Background(), rf, src)
	require.ErrorContains(t, err, "bad")
	require.Empty(t, fb.deployed, "the first release must not deploy when a later one fails to plan")
}

// A failure mid-apply stops immediately and still reports what was done.
func TestApplyFailsFastAndReportsPartialProgress(t *testing.T) {
	body := `releases:
  - name: first
    chart: swarmcli-charts/demo
    version: "0.1.0"
  - name: second
    chart: swarmcli-charts/demo
    version: "0.1.0"
`
	e, fb, src, rf := applyEnv(t, body, "0.1.0")
	ctx := context.Background()

	plan, err := e.PlanApply(ctx, rf, src)
	require.NoError(t, err)

	fb.failNext = true // first DeployStack fails
	res, err := e.Apply(ctx, plan, InstallOptions{})
	require.Error(t, err)
	require.ErrorContains(t, err, "first")
	require.Empty(t, res, "nothing completed")
	require.NotContains(t, fb.deployed, "second", "apply must stop at the first failure")
}

// A release that appears between planning and applying must upgrade cleanly
// rather than colliding with "already exists".
func TestApplyToleratesReleaseAppearingAfterPlan(t *testing.T) {
	e, fb, src, rf := applyEnv(t, oneRelease, "0.1.0")
	ctx := context.Background()

	plan, err := e.PlanApply(ctx, rf, src)
	require.NoError(t, err)
	require.Equal(t, ActionInstall, plan.Releases[0].Action)

	// Someone else installs it in the gap.
	ch := demoChart(t, "0.1.0")
	_, err = e.Install(ctx, "hello", ReleaseChartOf(ch), ch.Values, "services: {}\n", InstallOptions{})
	require.NoError(t, err)

	_, err = e.Apply(ctx, plan, InstallOptions{})
	require.NoError(t, err, "apply upgrades over a release that appeared after planning")
	require.Contains(t, fb.deployed, "hello")
}

// Render must be byte-stable for identical inputs: the no-op detection compares
// rendered manifests, and `charts diff upgrade` already relies on it.
func TestRenderIsDeterministic(t *testing.T) {
	ch := demoChart(t, "0.1.0")
	ctx := RenderContext{
		Values:  ch.Values,
		Release: ReleaseMeta{Name: "hello", Namespace: "hello", Revision: 1},
		Chart:   ChartMeta{Name: ch.Metadata.Name, Version: "0.1.0"},
	}
	first, err := Render(ch, ctx)
	require.NoError(t, err)
	for range 20 {
		again, err := Render(ch, ctx)
		require.NoError(t, err)
		require.Equal(t, first, again, "Render must be byte-stable or every apply churns a revision")
	}
}

func TestPlanCounts(t *testing.T) {
	p := &Plan{Releases: []ReleasePlan{
		{Action: ActionInstall}, {Action: ActionUpgrade},
		{Action: ActionUnchanged}, {Action: ActionUnchanged},
	}}
	i, u, n := p.Counts()
	require.Equal(t, 1, i)
	require.Equal(t, 1, u)
	require.Equal(t, 2, n)
}
