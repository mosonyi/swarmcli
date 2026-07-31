// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
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

// Planning records a chart's engine requirement but must not act on it: apply
// plans every release before converging any, so the whole plan is gated at once
// by the caller.
func TestPlanApplyRecordsCompatFinding(t *testing.T) {
	withEngineVersion(t, "1.12.0")
	e, _, src, rf := applyEnv(t, oneRelease)

	ch := demoChart(t, "0.1.0")
	ch.Metadata.SwarmcliVersion = ">= 1.13.0"
	src.charts["swarmcli-charts/demo@0.1.0"] = ch

	plan, err := e.PlanApply(context.Background(), rf, src, PlanOptions{})
	require.NoError(t, err, "an incompatible chart must still plan; refusing is the caller's call")
	require.Len(t, plan.Releases, 1)
	require.Equal(t, ActionInstall, plan.Releases[0].Action)
	require.Equal(t, CompatIncompatible, plan.Releases[0].Compat.Status)
	require.Equal(t, ">= 1.13.0", plan.Releases[0].Compat.Required)
}

// The motivating case: a chart needing a newer engine usually dies inside Render
// long before the caller's gate sees the finding, so the render error itself has
// to carry the diagnosis.
func TestPlanApplyRenderErrorNamesTheEngineRequirement(t *testing.T) {
	withEngineVersion(t, "1.12.0")
	e, _, src, rf := applyEnv(t, oneRelease)

	broken := demoChart(t, "0.1.0")
	broken.Metadata.SwarmcliVersion = ">= 1.13.0"
	broken.Templates = map[string]string{"templates/stack.yaml": "{{ toYamlPretty .Values }}"}
	src.charts["swarmcli-charts/demo@0.1.0"] = broken

	_, err := e.PlanApply(context.Background(), rf, src, PlanOptions{})
	require.Error(t, err)
	require.ErrorContains(t, err, "toYamlPretty", "the underlying failure must survive")
	require.ErrorContains(t, err, "requires swarmcli >= 1.13.0", "and be diagnosed")
	require.ErrorContains(t, err, "likely a consequence")
}

// The same failure on a chart that declared nothing must not acquire a spurious
// compatibility story.
func TestPlanApplyRenderErrorUnannotatedWhenNothingDeclared(t *testing.T) {
	withEngineVersion(t, "1.12.0")
	e, _, src, rf := applyEnv(t, oneRelease)

	broken := demoChart(t, "0.1.0")
	broken.Templates = map[string]string{"templates/stack.yaml": "{{ toYamlPretty .Values }}"}
	src.charts["swarmcli-charts/demo@0.1.0"] = broken

	_, err := e.PlanApply(context.Background(), rf, src, PlanOptions{})
	require.Error(t, err)
	require.ErrorContains(t, err, "toYamlPretty")
	require.NotContains(t, err.Error(), "likely a consequence")
}

func TestApplyInstallsThenIsIdempotent(t *testing.T) {
	e, fb, src, rf := applyEnv(t, oneRelease, "0.1.0")
	ctx := context.Background()

	plan, err := e.PlanApply(ctx, rf, src, PlanOptions{})
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

	plan2, err := e.PlanApply(ctx, rf, src, PlanOptions{})
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

	plan, err := e.PlanApply(ctx, rf, src, PlanOptions{})
	require.NoError(t, err)
	_, err = e.Apply(ctx, plan, InstallOptions{})
	require.NoError(t, err)

	// This is exactly what Renovate does to the file: bump the pinned version.
	bumped := &ReleaseFile{
		Path: rf.Path, Dir: rf.Dir,
		Releases: []ReleaseSpec{{Name: "hello", Chart: "swarmcli-charts/demo", Version: "0.2.0"}},
	}
	src.charts["swarmcli-charts/demo@0.2.0"] = demoChart(t, "0.2.0")

	plan2, err := e.PlanApply(ctx, bumped, src, PlanOptions{})
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

	plan, err := e.PlanApply(ctx, rf, src, PlanOptions{})
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

	plan2, err := e.PlanApply(ctx, withValues, src, PlanOptions{})
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

	plan, err := e.PlanApply(ctx, rf, src, PlanOptions{})
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

	_, err := e.PlanApply(context.Background(), rf, src, PlanOptions{})
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

	plan, err := e.PlanApply(ctx, rf, src, PlanOptions{})
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

	plan, err := e.PlanApply(ctx, rf, src, PlanOptions{})
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

// The repository path of apply — repositories: -> EnsureRepos -> Resolve -> Pull
// -> render -> plan — was covered by NOTHING at any tier: the unit tests use a
// fakeChartSource that never touches RepoStore, and the integration test passes a
// nil store with a local chart dir. This is the only test that would catch a break
// anywhere along the chain a downstream user actually runs.
func TestPlanApplyAgainstARealRepository(t *testing.T) {
	store := serveRepo(t, "0.1.0", "0.2.0")
	repos, err := store.List()
	require.NoError(t, err)
	url := repos[0].URL

	dir := t.TempDir()
	path := filepath.Join(dir, "swarmcli-release.yaml")
	require.NoError(t, os.WriteFile(path, []byte(
		"repositories:\n  - name: eldara\n    url: "+url+"\n"+
			"releases:\n  - name: hello\n    chart: eldara/demo\n    version: \"0.1.0\"\n"), 0o600))

	rf, err := LoadReleaseFile(path)
	require.NoError(t, err)

	// A fresh store, so EnsureRepos genuinely has to add the repository — exactly
	// what `charts apply -f` does on a CI runner that has never seen it.
	fresh := newTestStore(t)
	require.NoError(t, fresh.EnsureRepos(rf.Repositories))

	e := NewEngineWith(newFakeBackend())
	plan, err := e.PlanApply(context.Background(), rf, NewChartSource(fresh), PlanOptions{})
	require.NoError(t, err)
	require.Len(t, plan.Releases, 1)
	require.Equal(t, ActionInstall, plan.Releases[0].Action)
	require.Equal(t, "0.1.0", plan.Releases[0].ToVersion, "the PINNED version must win, not the latest")
	require.NotEmpty(t, plan.Releases[0].Manifest)

	res, err := e.Apply(context.Background(), plan, InstallOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, res[0].Revision)
}

// A release row left in StatusUninstalled must plan as an install, not silently
// no-op as "unchanged".
func TestPlanApplyReinstallsAnUninstalledRelease(t *testing.T) {
	e, fb, src, rf := applyEnv(t, oneRelease, "0.1.0")
	ctx := context.Background()

	plan, err := e.PlanApply(ctx, rf, src, PlanOptions{})
	require.NoError(t, err)
	_, err = e.Apply(ctx, plan, InstallOptions{})
	require.NoError(t, err)

	_, err = e.Uninstall(ctx, "hello", false)
	require.NoError(t, err)
	require.NotContains(t, fb.deployed, "hello")

	plan2, err := e.PlanApply(ctx, rf, src, PlanOptions{})
	require.NoError(t, err)
	require.Equal(t, ActionInstall, plan2.Releases[0].Action,
		"an uninstalled release must be reinstalled, not reported unchanged")

	_, err = e.Apply(ctx, plan2, InstallOptions{})
	require.NoError(t, err)
	require.Contains(t, fb.deployed, "hello")
}

// Same chart version and same values, but the chart's template changed: the
// rendered manifest differs, so it is an upgrade. Nothing exercised the manifest
// half of the comparison on its own.
func TestPlanApplyDetectsManifestOnlyChange(t *testing.T) {
	e, _, src, rf := applyEnv(t, oneRelease, "0.1.0")
	ctx := context.Background()

	plan, err := e.PlanApply(ctx, rf, src, PlanOptions{})
	require.NoError(t, err)
	_, err = e.Apply(ctx, plan, InstallOptions{})
	require.NoError(t, err)

	// Re-publish the SAME chart version with a different template body. The edit
	// has to survive rendering: Render deep-merges and re-marshals the Compose
	// document, so an added comment would be dropped and the manifest would come
	// out byte-identical.
	edited := demoChart(t, "0.1.0")
	edited.Templates["stack.yaml"] += "    stop_grace_period: 42s\n"
	src.charts["swarmcli-charts/demo@0.1.0"] = edited

	plan2, err := e.PlanApply(ctx, rf, src, PlanOptions{})
	require.NoError(t, err)
	require.Equal(t, ActionUpgrade, plan2.Releases[0].Action)
}

// A missing values file must abort planning, before anything deploys, naming the
// path.
func TestPlanApplyMissingValuesFileFailsBeforeDeploying(t *testing.T) {
	body := `releases:
  - name: hello
    chart: swarmcli-charts/demo
    version: "0.1.0"
    values: [./absent.yaml]
`
	e, fb, src, rf := applyEnv(t, body, "0.1.0")

	_, err := e.PlanApply(context.Background(), rf, src, PlanOptions{})
	require.ErrorContains(t, err, "absent.yaml")
	require.ErrorContains(t, err, "hello")
	require.Empty(t, fb.deployed)
}

// The file path and release name in the error are the whole ergonomic point of the
// planRelease wrappers.
func TestPlanApplyErrorsNameTheFileAndRelease(t *testing.T) {
	e, _, src, rf := applyEnv(t, oneRelease, "0.1.0")

	// A chart version the source cannot serve.
	rf.Releases[0].Version = "9.9.9"
	_, err := e.PlanApply(context.Background(), rf, src, PlanOptions{})
	require.ErrorContains(t, err, rf.Path)
	require.ErrorContains(t, err, `release "hello"`)
}

// A values file that violates the chart's schema must abort planning — before any
// release deploys — with the file and release named. This is the wrapper that
// makes a bad value in one release safe for all the others.
func TestPlanApplyRejectsSchemaViolationBeforeDeploying(t *testing.T) {
	e, fb, src, rf := applyEnv(t, oneRelease, "0.1.0")

	bad := filepath.Join(rf.Dir, "bad.yaml")
	require.NoError(t, os.WriteFile(bad, []byte("replicas: 0\n"), 0o600)) // schema: minimum 1
	rf.Releases[0].Values = []string{"./bad.yaml"}

	_, err := e.PlanApply(context.Background(), rf, src, PlanOptions{})
	require.Error(t, err)
	require.ErrorContains(t, err, rf.Path)
	require.ErrorContains(t, err, `release "hello"`)
	require.Empty(t, fb.deployed, "a schema violation must abort before anything deploys")
}

// Pointing a release at a different chart entirely (not just a new version) is an
// upgrade, not "unchanged".
func TestPlanApplyDetectsChartNameChange(t *testing.T) {
	e, _, src, rf := applyEnv(t, oneRelease, "0.1.0")
	ctx := context.Background()

	plan, err := e.PlanApply(ctx, rf, src, PlanOptions{})
	require.NoError(t, err)
	_, err = e.Apply(ctx, plan, InstallOptions{})
	require.NoError(t, err)

	other := demoChart(t, "0.1.0")
	other.Metadata.Name = "different"
	src.charts["swarmcli-charts/other@0.1.0"] = other
	rf.Releases[0].Chart = "swarmcli-charts/other"

	plan2, err := e.PlanApply(ctx, rf, src, PlanOptions{})
	require.NoError(t, err)
	require.Equal(t, ActionUpgrade, plan2.Releases[0].Action)
}

// ownedRelease is a release file that claims what it installs.
const ownedRelease = `owner: prod-swarm
repositories:
  - name: swarmcli-charts
    url: https://eldara-tech.github.io/swarmcli-charts
releases:
  - name: hello
    chart: swarmcli-charts/demo
    version: "0.1.0"
`

// storeRelease writes a release record straight into the backend, so a test can
// stage history that the engine would not produce itself — in particular a stamp
// naming a release other than the one carrying it.
func storeRelease(t *testing.T, fb *fakeBackend, rel Release) {
	t.Helper()
	payload, err := yaml.Marshal(rel)
	require.NoError(t, err)
	gz, err := gzipBytes(payload)
	require.NoError(t, err)
	labels := map[string]string{LabelType: TypeRelease, LabelRelease: rel.Name, LabelStatus: rel.Status}
	if rel.Owner != "" {
		labels[LabelOwner] = rel.Owner
	}
	require.NoError(t, fb.CreateConfig(context.Background(), releaseConfigName(rel.Name, rel.Revision), gz, labels))
}

// A release this file installed and no longer declares is provably obsolete: the
// stamp names this manifest, so nothing else can be claiming it. That is the
// distinction the stamp exists to draw, and it is the whole prerequisite for a
// prune — so it must not land in Unmanaged alongside releases of unknown origin.
func TestPlanApplySeparatesItsOwnOrphansFromUnmanagedReleases(t *testing.T) {
	e, fb, src, rf := applyEnv(t, ownedRelease, "0.1.0")
	ctx := context.Background()

	storeRelease(t, fb, Release{Name: "ours", Revision: 1, Status: StatusDeployed, Owner: "apply/prod-swarm:release/ours"})
	storeRelease(t, fb, Release{Name: "theirs", Revision: 1, Status: StatusDeployed, Owner: "apply/other-swarm:release/theirs"})
	storeRelease(t, fb, Release{Name: "byhand", Revision: 1, Status: StatusDeployed})

	plan, err := e.PlanApply(ctx, rf, src, PlanOptions{})
	require.NoError(t, err)
	require.Equal(t, "apply/prod-swarm", plan.Owner)
	require.Equal(t, []string{"ours"}, plan.Orphaned)
	require.ElementsMatch(t, []string{"byhand", "theirs"}, plan.Unmanaged)
}

// The stamp names the resource it was written for, which is the point of
// encoding a tuple rather than a bare owner string: a record copied onto another
// release still names the original, so it is not evidence that this manifest
// installed the copy. ArgoCD's original bare instance label could not tell those
// apart, and this is the case that proves ours can.
func TestPlanApplyIgnoresAStampNamingAnotherRelease(t *testing.T) {
	e, fb, src, rf := applyEnv(t, ownedRelease, "0.1.0")

	// Right owner, wrong resource — a copy of "ours" filed under a new name.
	storeRelease(t, fb, Release{Name: "copy", Revision: 1, Status: StatusDeployed, Owner: "apply/prod-swarm:release/ours"})
	// Right owner, unparseable stamp.
	storeRelease(t, fb, Release{Name: "garbled", Revision: 1, Status: StatusDeployed, Owner: "apply/prod-swarm"})

	plan, err := e.PlanApply(context.Background(), rf, src, PlanOptions{})
	require.NoError(t, err)
	require.Empty(t, plan.Orphaned, "an unverifiable stamp is not ownership")
	require.ElementsMatch(t, []string{"copy", "garbled"}, plan.Unmanaged)
}

// A file that declares no owner claims nothing, so every release on the swarm
// stays unmanaged no matter what stamp it carries. That is what keeps the stamp
// opt-in and today's behaviour the default.
func TestPlanApplyWithoutAnOwnerClaimsNothing(t *testing.T) {
	e, fb, src, rf := applyEnv(t, oneRelease, "0.1.0")

	storeRelease(t, fb, Release{Name: "stamped", Revision: 1, Status: StatusDeployed, Owner: "apply/prod-swarm:release/stamped"})

	plan, err := e.PlanApply(context.Background(), rf, src, PlanOptions{})
	require.NoError(t, err)
	require.Empty(t, plan.Owner)
	require.Empty(t, plan.Orphaned)
	require.Equal(t, []string{"stamped"}, plan.Unmanaged)
}

// Applying a plan stamps what it installs with the owner the plan was
// classified against, so the next run recognises its own work.
func TestApplyStampsTheOwnerItPlannedWith(t *testing.T) {
	e, _, src, rf := applyEnv(t, ownedRelease, "0.1.0")
	ctx := context.Background()

	plan, err := e.PlanApply(ctx, rf, src, PlanOptions{})
	require.NoError(t, err)
	_, err = e.Apply(ctx, plan, InstallOptions{Owner: plan.Owner})
	require.NoError(t, err)

	rel, err := e.GetRevision(ctx, "hello", 0)
	require.NoError(t, err)
	require.Equal(t, "apply/prod-swarm:release/hello", rel.Owner)

	// Drop it from the file — same owner, same swarm — and what apply installed,
	// apply now recognises as its own orphan rather than as a stranger.
	_, _, src2, rf2 := applyEnv(t, `owner: prod-swarm
releases:
  - name: other
    chart: swarmcli-charts/demo
    version: "0.1.0"
`, "0.1.0")
	plan2, err := e.PlanApply(ctx, rf2, src2, PlanOptions{})
	require.NoError(t, err)
	require.Equal(t, []string{"hello"}, plan2.Orphaned)
}

// apply still deletes nothing, orphan or not: the stamp establishes that it
// could, and acting on it is a separate change.
func TestApplyDoesNotRemoveItsOwnOrphans(t *testing.T) {
	e, fb, src, rf := applyEnv(t, ownedRelease, "0.1.0")
	ctx := context.Background()

	ch := demoChart(t, "0.1.0")
	_, err := e.Install(ctx, "obsolete", ReleaseChartOf(ch), ch.Values, "services: {}\n",
		InstallOptions{Owner: "apply/prod-swarm"})
	require.NoError(t, err)

	plan, err := e.PlanApply(ctx, rf, src, PlanOptions{})
	require.NoError(t, err)
	require.Equal(t, []string{"obsolete"}, plan.Orphaned)

	_, err = e.Apply(ctx, plan, InstallOptions{Owner: plan.Owner})
	require.NoError(t, err)
	require.Contains(t, fb.deployed, "obsolete")
	require.Contains(t, fb.configs, "swarmcli.release.obsolete.v1")
}

// A controller installs under an id of its own — "cd/<app>" — and must be able to
// plan against that same id. Deriving "apply/<owner>" from the file regardless
// classified the controller's own releases as Unmanaged from the first reconcile:
// "I do not recognise this", about releases this exact caller installed.
func TestPlanApplyClassifiesAgainstTheOwnerTheCallerSupplied(t *testing.T) {
	e, fb, src, rf := applyEnv(t, ownedRelease, "0.1.0")
	ctx := context.Background()

	storeRelease(t, fb, Release{Name: "ours", Revision: 1, Status: StatusDeployed, Owner: "cd/edge:release/ours"})
	// What the file would have claimed, had the caller not overridden it.
	storeRelease(t, fb, Release{Name: "theirs", Revision: 1, Status: StatusDeployed, Owner: "apply/prod-swarm:release/theirs"})

	plan, err := e.PlanApply(ctx, rf, src, PlanOptions{Owner: "cd/edge"})
	require.NoError(t, err)
	require.Equal(t, "cd/edge", plan.Owner)
	require.Equal(t, []string{"ours"}, plan.Orphaned)
	require.Equal(t, []string{"theirs"}, plan.Unmanaged,
		"the file's own owner must not keep claiming once a caller has supplied one")
}

// The supplied id goes on to be stamped through Plan.Owner, so a plan and the
// apply that follows it agree — the whole point of classifying against it.
func TestPlanApplyRoundTripsTheSuppliedOwnerThroughApply(t *testing.T) {
	e, _, src, rf := applyEnv(t, oneRelease, "0.1.0")
	ctx := context.Background()

	plan, err := e.PlanApply(ctx, rf, src, PlanOptions{Owner: "cd/edge"})
	require.NoError(t, err)
	_, err = e.Apply(ctx, plan, InstallOptions{Owner: plan.Owner})
	require.NoError(t, err)

	// Drop it from the file; the next plan recognises its own work as an orphan
	// rather than as a stranger, which is what makes a later prune safe.
	_, _, src2, rf2 := applyEnv(t, `releases:
  - name: other
    chart: swarmcli-charts/demo
    version: "0.1.0"
`, "0.1.0")
	plan2, err := e.PlanApply(ctx, rf2, src2, PlanOptions{Owner: "cd/edge"})
	require.NoError(t, err)
	require.Equal(t, []string{"hello"}, plan2.Orphaned)
}

// ":" separates the id from the resource half, so an id containing one would
// decode as a different owner than it was written as. Install rejects it; so must
// planning, and before reading anything off the swarm.
func TestPlanApplyRejectsAnOwnerThatWouldNotSurviveTheEncoding(t *testing.T) {
	e, _, src, rf := applyEnv(t, oneRelease, "0.1.0")

	_, err := e.PlanApply(context.Background(), rf, src, PlanOptions{Owner: "cd:edge"})
	require.ErrorContains(t, err, "must not contain ':'")
}

// The reader sees the bytes between "this path was named" and "these values were
// merged", which is where a values file committed encrypted becomes plaintext
// without ever reaching the controller's filesystem.
func TestPlanApplyMergesValuesThroughTheSuppliedReader(t *testing.T) {
	body := `releases:
  - name: hello
    chart: swarmcli-charts/demo
    version: "0.1.0"
    values: [./secret.yaml]
`
	e, _, src, rf := applyEnv(t, body, "0.1.0")
	require.NoError(t, os.WriteFile(filepath.Join(rf.Dir, "secret.yaml"), []byte("ciphertext\n"), 0o600))

	var seen []string
	plan, err := e.PlanApply(context.Background(), rf, src, PlanOptions{
		ReadFile: func(path string) ([]byte, error) {
			seen = append(seen, path)
			return []byte("injected: 1\n"), nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, []string{filepath.Join(rf.Dir, "secret.yaml")}, seen,
		"the reader is given the resolved path, so it can decide by name")
	require.Equal(t, 1, plan.Releases[0].Values["injected"])
}

// A reader that fails aborts the whole plan, exactly as an unreadable file does:
// nothing is deployed on the strength of values that could not be resolved.
func TestPlanApplyReaderErrorFailsBeforeDeploying(t *testing.T) {
	body := `releases:
  - name: hello
    chart: swarmcli-charts/demo
    version: "0.1.0"
    values: [./secret.yaml]
`
	e, fb, src, rf := applyEnv(t, body, "0.1.0")

	_, err := e.PlanApply(context.Background(), rf, src, PlanOptions{
		ReadFile: func(string) ([]byte, error) { return nil, errors.New("no decryption key") },
	})
	require.ErrorContains(t, err, "no decryption key")
	require.ErrorContains(t, err, "hello")
	require.Empty(t, fb.deployed)
}
