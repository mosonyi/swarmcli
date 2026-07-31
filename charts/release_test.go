// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// fakeBackend is an in-memory Backend for lifecycle tests.
type fakeBackend struct {
	configs       map[string]fakeConfig
	deployed      map[string]string // stack name -> manifest
	volumes       map[string][]string
	services      map[string][]ServiceState
	networkScopes map[string]string     // network name -> scope
	secrets       map[string]struct{}   // existing secret names
	createNetErr  map[string]error      // network name -> error to return on create
	createdNets   map[string]createdNet // network name -> driver/attachable used on create
	failNext      bool
	rmStackErr    error
	refreshErr    error
	secretsErr    error                   // error to return from SecretNames
	onCreate      func(name string) error // hook to simulate concurrent config creation
	deleteCfgErr  map[string]error        // config name -> error to return on delete
	// listData makes ListConfigs carry each payload, as the Docker backend
	// does. Off by default so the rest of the suite keeps exercising the
	// inspect fallback a Backend that omits it relies on.
	listData bool
	inspects int // InspectConfig calls, to prove the payload came from the list
}

type fakeConfig struct {
	data   []byte
	labels map[string]string
}

type createdNet struct {
	driver     string
	attachable bool
}

func newFakeBackend() *fakeBackend {
	return &fakeBackend{
		configs:       map[string]fakeConfig{},
		deployed:      map[string]string{},
		volumes:       map[string][]string{},
		services:      map[string][]ServiceState{},
		networkScopes: map[string]string{},
		secrets:       map[string]struct{}{},
		createNetErr:  map[string]error{},
	}
}

func (f *fakeBackend) DeployStack(_ context.Context, name, manifest, resolve string) error {
	if f.failNext {
		f.failNext = false
		return fmt.Errorf("boom")
	}
	f.deployed[name] = manifest
	return nil
}
func (f *fakeBackend) RemoveStack(_ context.Context, name string) error {
	if f.rmStackErr != nil {
		return f.rmStackErr
	}
	delete(f.deployed, name)
	return nil
}
func (f *fakeBackend) RefreshSnapshot(context.Context) error { return f.refreshErr }
func (f *fakeBackend) CreateConfig(_ context.Context, name string, data []byte, labels map[string]string) error {
	if f.onCreate != nil {
		if err := f.onCreate(name); err != nil {
			return err
		}
	}
	if _, ok := f.configs[name]; ok {
		return fmt.Errorf("config %q already exists", name)
	}
	f.configs[name] = fakeConfig{data: data, labels: labels}
	return nil
}
func (f *fakeBackend) ListConfigs(context.Context) ([]ConfigMeta, error) {
	var out []ConfigMeta
	for name, c := range f.configs {
		m := ConfigMeta{Name: name, Labels: c.labels}
		if f.listData {
			m.Data = c.data
		}
		out = append(out, m)
	}
	return out, nil
}
func (f *fakeBackend) InspectConfig(_ context.Context, name string) ([]byte, error) {
	f.inspects++
	c, ok := f.configs[name]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return c.data, nil
}
func (f *fakeBackend) DeleteConfig(_ context.Context, name string) error {
	if err := f.deleteCfgErr[name]; err != nil {
		return err
	}
	delete(f.configs, name)
	return nil
}
func (f *fakeBackend) StackServices(_ context.Context, name string) []ServiceState {
	return f.services[name]
}
func (f *fakeBackend) StackVolumes(_ context.Context, name string) ([]string, error) {
	return f.volumes[name], nil
}
func (f *fakeBackend) RemoveVolume(_ context.Context, name string) error {
	for stack, vols := range f.volumes {
		out := vols[:0]
		for _, v := range vols {
			if v != name {
				out = append(out, v)
			}
		}
		f.volumes[stack] = out
	}
	return nil
}

func (f *fakeBackend) NetworkScopes(context.Context) (map[string]string, error) {
	out := map[string]string{}
	for k, v := range f.networkScopes {
		out[k] = v
	}
	return out, nil
}
func (f *fakeBackend) CreateOverlayNetwork(_ context.Context, name, driver string, attachable bool) error {
	if err := f.createNetErr[name]; err != nil {
		return err
	}
	if f.createdNets == nil {
		f.createdNets = map[string]createdNet{}
	}
	f.createdNets[name] = createdNet{driver: driver, attachable: attachable}
	f.networkScopes[name] = "swarm"
	return nil
}
func (f *fakeBackend) RemoveOverlayNetwork(_ context.Context, name string) error {
	delete(f.networkScopes, name)
	return nil
}
func (f *fakeBackend) SecretNames(context.Context) (map[string]struct{}, error) {
	if f.secretsErr != nil {
		return nil, f.secretsErr
	}
	out := make(map[string]struct{}, len(f.secrets))
	for k := range f.secrets {
		out[k] = struct{}{}
	}
	return out, nil
}

func testEngine(b Backend) *Engine {
	e := NewEngineWith(b)
	e.now = func() time.Time { return time.Unix(1700000000, 0).UTC() }
	return e
}

func TestInstallRecordsRevisionOne(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	ctx := context.Background()

	rel, err := e.Install(ctx, "my-demo", ReleaseChart{Name: "demo", Version: "0.1.0"},
		map[string]any{"replicas": 1}, "version: \"3.9\"\nservices:\n  app:\n    image: x\n", InstallOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, rel.Revision)
	require.Equal(t, StatusDeployed, rel.Status)
	require.Equal(t, rel.Manifest, fb.deployed["my-demo"])

	cfg := fb.configs[releaseConfigName("my-demo", 1)]
	require.Equal(t, TypeRelease, cfg.labels[LabelType])
	require.Equal(t, "my-demo", cfg.labels[LabelRelease])
	require.Equal(t, "1", cfg.labels[LabelRevision])
}

func TestInstallRejectsExisting(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	ctx := context.Background()
	_, err := e.Install(ctx, "demo", ReleaseChart{Name: "demo", Version: "1"}, nil, "services:\n  a:\n    image: x\n", InstallOptions{})
	require.NoError(t, err)
	_, err = e.Install(ctx, "demo", ReleaseChart{Name: "demo", Version: "1"}, nil, "services:\n  a:\n    image: x\n", InstallOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "already exists")
}

func TestInstallDryRunDoesNotDeploy(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	rel, err := e.Install(context.Background(), "demo", ReleaseChart{Name: "demo", Version: "1"}, nil, "services:\n  a:\n    image: x\n", InstallOptions{DryRun: true})
	require.NoError(t, err)
	require.Equal(t, 1, rel.Revision)
	require.Empty(t, fb.deployed)
	require.Empty(t, fb.configs)
}

// A failed deploy must record nothing, so the install stays retryable rather
// than leaving an orphaned "already exists" release Config behind.
func TestInstallFailureDoesNotRecord(t *testing.T) {
	fb := newFakeBackend()
	fb.failNext = true
	e := testEngine(fb)
	manifest := "services:\n  a:\n    image: x\n"

	rel, err := e.Install(context.Background(), "demo", ReleaseChart{Name: "demo", Version: "1"}, nil, manifest, InstallOptions{})
	require.Error(t, err)
	require.Equal(t, StatusFailed, rel.Status)
	require.Empty(t, fb.configs, "failed deploy must not persist a release Config")

	// failNext cleared itself; a retry now succeeds at revision 1.
	rel, err = e.Install(context.Background(), "demo", ReleaseChart{Name: "demo", Version: "1"}, nil, manifest, InstallOptions{})
	require.NoError(t, err)
	require.Equal(t, 1, rel.Revision)
	require.Equal(t, StatusDeployed, rel.Status)
}

const extNetManifest = "version: \"3.9\"\n" +
	"services:\n  app:\n    image: x\n" +
	"networks:\n  traefik-public:\n    external: true\n"

func TestInstallExternalNetworkPresent(t *testing.T) {
	fb := newFakeBackend()
	fb.networkScopes["traefik-public"] = "swarm"
	e := testEngine(fb)
	rel, err := e.Install(context.Background(), "demo", ReleaseChart{Name: "demo", Version: "1"}, nil, extNetManifest, InstallOptions{})
	require.NoError(t, err)
	require.Equal(t, StatusDeployed, rel.Status)
	require.Equal(t, extNetManifest, fb.deployed["demo"])
}

// A deploy failure must roll back any network this install auto-created, so a
// failed install leaves no trace behind.
func TestInstallRollsBackAutoCreatedNetworkOnDeployFailure(t *testing.T) {
	fb := newFakeBackend()
	fb.failNext = true // DeployStack fails
	e := testEngine(fb)
	_, err := e.Install(context.Background(), "demo", ReleaseChart{Name: "demo", Version: "1"}, nil, extNetManifest, InstallOptions{})
	require.Error(t, err)
	require.NotContains(t, fb.networkScopes, "traefik-public")
}

func TestInstallExternalNetworkAutoCreated(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	rel, err := e.Install(context.Background(), "demo", ReleaseChart{Name: "demo", Version: "1"}, nil, extNetManifest, InstallOptions{})
	require.NoError(t, err)
	require.Equal(t, StatusDeployed, rel.Status)
	require.Equal(t, "swarm", fb.networkScopes["traefik-public"], "missing external network should be auto-created swarm-scoped")
	require.Equal(t, extNetManifest, fb.deployed["demo"])
}

func TestInstallExternalNetworkCreateFails(t *testing.T) {
	fb := newFakeBackend()
	fb.createNetErr["traefik-public"] = fmt.Errorf("not a swarm manager")
	e := testEngine(fb)
	rel, err := e.Install(context.Background(), "demo", ReleaseChart{Name: "demo", Version: "1"}, nil, extNetManifest, InstallOptions{})
	require.Error(t, err)
	require.Equal(t, StatusFailed, rel.Status)
	require.Contains(t, err.Error(), "docker network create --driver overlay --attachable traefik-public")
	require.Empty(t, fb.deployed, "deploy must not run when an external network is unavailable")
	require.Empty(t, fb.configs, "nothing should be recorded")
}

func TestInstallExternalNetworkScopeClash(t *testing.T) {
	fb := newFakeBackend()
	fb.networkScopes["traefik-public"] = "local"
	e := testEngine(fb)
	_, err := e.Install(context.Background(), "demo", ReleaseChart{Name: "demo", Version: "1"}, nil, extNetManifest, InstallOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), "non-swarm")
	require.Empty(t, fb.deployed)
	require.Empty(t, fb.configs)
}

func TestListReturnsCurrentRevisions(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	ctx := context.Background()
	_, _ = e.Install(ctx, "a", ReleaseChart{Name: "ca", Version: "1"}, nil, "services:\n  s:\n    image: x\n", InstallOptions{})
	_, _ = e.Install(ctx, "b", ReleaseChart{Name: "cb", Version: "1"}, nil, "services:\n  s:\n    image: x\n", InstallOptions{})

	list, err := e.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
	require.Equal(t, "a", list[0].Name)
	require.Equal(t, "b", list[1].Name)
}

// Reading release history is the engine's hottest path — every List, Status,
// Install, Upgrade, Rollback and GetRevision walks it, and a GitOps controller
// walks it on a timer. When the Backend's listing carries the payload, that walk
// is one API call rather than one per stored revision (issue #510).
func TestAllRevisionsReadsThePayloadFromTheListing(t *testing.T) {
	fb := newFakeBackend()
	fb.listData = true
	e := testEngine(fb)
	ctx := context.Background()
	_, err := e.Install(ctx, "a", ReleaseChart{Name: "ca", Version: "1"}, nil, "services:\n  s:\n    image: x\n", InstallOptions{})
	require.NoError(t, err)
	_, err = e.Upgrade(ctx, "a", ReleaseChart{Name: "ca", Version: "2"}, nil, "services:\n  s:\n    image: y\n", InstallOptions{})
	require.NoError(t, err)

	fb.inspects = 0
	list, err := e.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, 2, list[0].Revision, "the payload from the listing must decode to the same release")
	require.Zero(t, fb.inspects, "the payload was in the listing; inspecting again is a wasted round trip per revision")
}

// A Backend that lists names and labels but not payloads stays correct: the
// field is additive, so an implementation written before it existed keeps
// working through the inspect it always did.
func TestAllRevisionsInspectsWhenTheListingOmitsThePayload(t *testing.T) {
	fb := newFakeBackend() // listData off
	e := testEngine(fb)
	ctx := context.Background()
	_, err := e.Install(ctx, "a", ReleaseChart{Name: "ca", Version: "1"}, nil, "services:\n  s:\n    image: x\n", InstallOptions{})
	require.NoError(t, err)

	fb.inspects = 0
	list, err := e.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, 1, list[0].Revision)
	require.Equal(t, 1, fb.inspects, "the payload was not in the listing, so it had to be inspected")
}

func TestUninstallRemovesStackAndConfigsKeepsVolumes(t *testing.T) {
	fb := newFakeBackend()
	fb.volumes["demo"] = []string{"demo_data"}
	e := testEngine(fb)
	ctx := context.Background()
	_, err := e.Install(ctx, "demo", ReleaseChart{Name: "demo", Version: "1"}, nil, "services:\n  s:\n    image: x\n", InstallOptions{})
	require.NoError(t, err)

	_, err = e.Uninstall(ctx, "demo", false)
	require.NoError(t, err)
	require.Empty(t, fb.deployed)
	require.Empty(t, fb.configs)
	require.Equal(t, []string{"demo_data"}, fb.volumes["demo"]) // volume retained

	_, err = e.Uninstall(ctx, "demo", false) // already gone
	require.Error(t, err)
}

func TestUninstallPurgeVolumes(t *testing.T) {
	fb := newFakeBackend()
	fb.volumes["demo"] = []string{"demo_data"}
	e := testEngine(fb)
	ctx := context.Background()
	_, _ = e.Install(ctx, "demo", ReleaseChart{Name: "demo", Version: "1"}, nil, "services:\n  s:\n    image: x\n", InstallOptions{})
	_, err := e.Uninstall(ctx, "demo", true)
	require.NoError(t, err)
	require.Empty(t, fb.volumes["demo"])
}

func TestStatusReturnsServices(t *testing.T) {
	fb := newFakeBackend()
	fb.services["demo"] = []ServiceState{{Name: "app", Mode: "replicated", Replicas: "1/1", Status: "running"}}
	e := testEngine(fb)
	ctx := context.Background()
	_, _ = e.Install(ctx, "demo", ReleaseChart{Name: "demo", Version: "1"}, nil, "services:\n  s:\n    image: x\n", InstallOptions{})
	rel, svcs, err := e.Status(ctx, "demo")
	require.NoError(t, err)
	require.Equal(t, 1, rel.Revision)
	require.Len(t, svcs, 1)
	require.Equal(t, "app", svcs[0].Name)
}

func mustGzipRelease(t *testing.T, rel *Release) []byte {
	t.Helper()
	payload, err := yaml.Marshal(rel)
	require.NoError(t, err)
	gz, err := gzipBytes(payload)
	require.NoError(t, err)
	return gz
}

// stableAge outlives any monitor window these tests declare, so a service
// carrying it has already served out its stability window and parity is the only
// thing left to decide.
const stableAge = time.Hour

func phaseOf(s ServiceState) Phase { return s.Convergence().Phase }

// A service at parity whose update is still in flight must not count as ready.
func TestConvergenceRejectsInProgress(t *testing.T) {
	require.Equal(t, PhaseConverged, phaseOf(ServiceState{Running: 2, Desired: 2, NewestTaskAge: stableAge}))
	require.Equal(t, PhaseConverged, phaseOf(ServiceState{Running: 1, Desired: 1, UpdateState: "completed", NewestTaskAge: stableAge}))
	require.Equal(t, PhaseProgressing, phaseOf(ServiceState{Running: 2, Desired: 2, UpdateState: "updating", NewestTaskAge: stableAge}))
	require.Equal(t, PhaseProgressing, phaseOf(ServiceState{Running: 1, Desired: 1, UpdateState: "rollback_started", NewestTaskAge: stableAge}))
	require.Equal(t, PhaseProgressing, phaseOf(ServiceState{Running: 1, Desired: 2, NewestTaskAge: stableAge}))
}

// The bug in #473: a fresh install has no UpdateStatus, so the in-flight guard
// cannot fire, and the old check compared a Replicas string counting tasks by
// DESIRED state — which reaches parity the moment swarm schedules them. A
// release whose tasks are all still pending must not read as converged.
func TestConvergenceRejectsScheduledButNotRunning(t *testing.T) {
	scheduled := ServiceState{
		Name:          "api",
		Replicas:      "3/3", // what the old check saw: 3 tasks DESIRED running
		Status:        "active",
		Running:       0, // ...none of which had actually started
		Desired:       3,
		UpdateState:   "", // fresh install: never updated
		NewestTaskAge: stableAge,
	}
	c := scheduled.Convergence()
	require.Equal(t, PhaseProgressing, c.Phase, "tasks that are merely scheduled must not count as converged")
	require.Equal(t, "0/3 tasks running", c.Reason)
}

// The bug in #443: a one-shot init or migration step ends with its task
// Complete and nothing running, which is its SUCCESS state. Requiring a running
// task held --wait until the timeout for a job that had already done its work,
// and showed the release as 0/1 forever.
func TestConvergenceAcceptsACompletedJob(t *testing.T) {
	done := ServiceState{
		Name:          "superset_init",
		Running:       0, // the task exited; swarm will not replace it
		Completed:     1,
		Desired:       1,
		Job:           true,
		NewestTaskAge: stableAge,
	}
	require.Equal(t, PhaseConverged, phaseOf(done), "a job that ran to completion is converged")

	// A job whose task has not finished yet is still not converged.
	require.Equal(t, PhaseProgressing, phaseOf(ServiceState{Running: 0, Completed: 0, Desired: 1, Job: true, NewestTaskAge: stableAge}))

	// The distinction that matters: a task that failed ends in state Failed, so
	// it is never counted as Completed and a broken job still blocks the wait.
	require.Equal(t, PhaseProgressing, phaseOf(ServiceState{Running: 0, Completed: 0, Desired: 1, Job: true, Status: "active", NewestTaskAge: stableAge}))
}

// A global service on a drained node lowers the target rather than leaving the
// release permanently short of a replica that can never be scheduled.
func TestConvergenceGlobalTracksActiveNodes(t *testing.T) {
	require.Equal(t, PhaseConverged, phaseOf(ServiceState{Mode: "global", Running: 2, Desired: 2, NewestTaskAge: stableAge}))
	require.Equal(t, PhaseProgressing, phaseOf(ServiceState{Mode: "global", Running: 1, Desired: 2, NewestTaskAge: stableAge}))
}

// Parity is necessary but not sufficient: a task that starts and then dies
// inside the monitor window is a rollout failure, so a service that has reached
// its target but not yet served out the window is still progressing.
func TestConvergenceHoldsUntilTheStabilityWindowCloses(t *testing.T) {
	c := ServiceState{Running: 1, Desired: 1, Monitor: 90 * time.Second, NewestTaskAge: 60 * time.Second}.Convergence()
	require.Equal(t, PhaseProgressing, c.Phase)
	require.Equal(t, "30s of the stability window remains", c.Reason)

	require.Equal(t, defaultStabilityWindow, ServiceState{Monitor: time.Second}.stabilityRemaining(),
		"a shorter monitor must not weaken the default")
}

// The window is measured from task creation, the same instant swarm measures it
// from — so a task that has already lived through it is stable NOW.
//
// This is the whole point. A task only reports running once its healthcheck
// passes, so start_period and the checks that followed are already spent by the
// time parity is observed. Restarting a full window at parity made --wait sit
// out the monitor period on top of however long the service took to come up,
// which is strictly more conservative than swarm and turns an honest
// `monitor: 8m` into an eight-minute install.
func TestStabilityRemainingCountsTimeTheTaskHasAlreadyLived(t *testing.T) {
	monitor := 90 * time.Second

	require.Equal(t, 30*time.Second,
		ServiceState{Monitor: monitor, NewestTaskAge: 60 * time.Second}.stabilityRemaining())
	require.LessOrEqual(t, ServiceState{Monitor: monitor, NewestTaskAge: monitor}.stabilityRemaining(),
		time.Duration(0), "a task that has outlived its monitor is already stable")
	require.LessOrEqual(t, ServiceState{Monitor: monitor, NewestTaskAge: 10 * time.Minute}.stabilityRemaining(),
		time.Duration(0), "an over-aged task must not produce a negative wait either")
}

// paused and rollback_paused are terminal: swarmkit never rolls back a
// rollback, so waiting out the timeout only delays the report.
func TestConvergenceWedged(t *testing.T) {
	require.Equal(t, PhaseProgressing, phaseOf(ServiceState{Name: "api", UpdateState: "updating"}))
	require.Equal(t, PhaseProgressing, phaseOf(ServiceState{Name: "api", UpdateState: ""}))

	c := Rollup([]ServiceState{{Name: "api", UpdateState: "paused"}})
	require.Equal(t, PhaseWedged, c.Phase)
	require.Contains(t, c.Reason, "api")

	c = Rollup([]ServiceState{{Name: "api", UpdateState: "rollback_paused"}})
	require.Equal(t, PhaseWedged, c.Phase)
	require.Contains(t, c.Reason, "manual recovery")
}

// The bug in #526: swarmkit restores the previous spec BEFORE it marks the
// rollback complete, so at "rollback_completed" the tasks that reach parity are
// running exactly what the deploy set out to replace. Falling through to the
// parity check let a failed rollout serve out the stability window and report
// converged — --wait returned nil and a pipeline gating on it went green.
func TestConvergenceRejectsACompletedRollback(t *testing.T) {
	rolledBack := ServiceState{Name: "api", Running: 1, Desired: 1, UpdateState: "rollback_completed", NewestTaskAge: stableAge}

	c := rolledBack.Convergence()
	require.Equal(t, PhaseWedged, c.Phase, "a deploy that was rolled back is a failed deploy, not a converged one")
	require.Contains(t, c.Reason, "rolled back")

	// And --wait says so instead of returning nil.
	prev := waitPollInterval
	waitPollInterval = time.Millisecond
	defer func() { waitPollInterval = prev }()

	b := &convergenceBackend{states: []ServiceState{rolledBack}}
	e := &Engine{Backend: b, now: time.Now}
	require.ErrorContains(t, e.waitReady(context.Background(), "rel", time.Hour), "rolled back")
	require.Equal(t, 1, b.calls, "should fail on the first observation, not wait out the timeout")
}

// The rollup is what a caller reads, so the phase that should worry it most has
// to win regardless of where in the list it appears — one wedged service must
// not be masked by another that is merely slow.
func TestRollupTakesTheWorstPhase(t *testing.T) {
	states := []ServiceState{
		{Name: "web", Running: 1, Desired: 1, NewestTaskAge: stableAge},
		{Name: "api", Running: 0, Desired: 1, NewestTaskAge: stableAge},
		{Name: "db", UpdateState: "paused"},
	}
	c := Rollup(states)
	require.Equal(t, PhaseWedged, c.Phase)
	require.Contains(t, c.Reason, `service "db"`)

	// Progressing wins over converged the same way, and names the service.
	c = Rollup(states[:2])
	require.Equal(t, PhaseProgressing, c.Phase)
	require.Equal(t, `service "api": 0/1 tasks running`, c.Reason)

	require.Equal(t, PhaseConverged, Rollup(states[:1]).Phase)
}

// A release whose stack reports nothing has not finished coming up. Reporting it
// converged would let --wait return the instant a deploy was accepted.
func TestRollupOfNoServicesIsProgressing(t *testing.T) {
	require.Equal(t, PhaseProgressing, Rollup(nil).Phase)
}

// When a concurrent actor claims the next revision number between read and
// write, record() must re-read the history and bump to the next free revision
// rather than colliding.
func TestRecordRetriesOnRevisionCollision(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	ctx := context.Background()

	_, err := e.Install(ctx, "demo", ReleaseChart{Name: "demo", Version: "1"}, nil, "services:\n  a:\n    image: x\n", InstallOptions{})
	require.NoError(t, err)

	// Simulate another actor recording v2 first: the create for v2 collides and
	// the colliding revision becomes visible, so the retry must allocate v3.
	fb.onCreate = func(name string) error {
		if name == releaseConfigName("demo", 2) {
			fb.onCreate = nil
			fb.configs[name] = fakeConfig{
				data:   mustGzipRelease(t, &Release{Name: "demo", Revision: 2, Status: StatusDeployed, Chart: ReleaseChart{Name: "demo", Version: "1"}}),
				labels: map[string]string{LabelType: TypeRelease, LabelRelease: "demo", LabelRevision: "2"},
			}
			return fmt.Errorf("config %q already exists", name)
		}
		return nil
	}
	rel, err := e.Upgrade(ctx, "demo", ReleaseChart{Name: "demo", Version: "2"}, nil, "services:\n  a:\n    image: y\n", InstallOptions{})
	require.NoError(t, err)
	require.Equal(t, 3, rel.Revision)
}

// A stack-removal failure during uninstall must not strand the release history:
// cleanup continues and the aggregated error is still reported.
func TestUninstallContinuesOnPartialFailure(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	ctx := context.Background()
	_, err := e.Install(ctx, "demo", ReleaseChart{Name: "demo", Version: "1"}, nil, "services:\n  a:\n    image: x\n", InstallOptions{})
	require.NoError(t, err)

	fb.rmStackErr = fmt.Errorf("stack gone")
	_, err = e.Uninstall(ctx, "demo", false)
	require.ErrorContains(t, err, "removing stack")
	require.Empty(t, fb.configs) // history cleaned up despite the stack-removal failure
}

// waitReady must hold parity for the stability window before reporting success:
// a task that starts and then dies inside UpdateConfig.Monitor is a rollout
// failure, so returning at first parity reports success for a service that is
// about to crash-loop.
func TestWaitReadyHoldsTheStabilityWindow(t *testing.T) {
	prev := waitPollInterval
	waitPollInterval = time.Millisecond
	defer func() { waitPollInterval = prev }()

	b := &convergenceBackend{states: []ServiceState{{Name: "api", Running: 1, Desired: 1}}}

	// The clock advances far too slowly to clear the 5s stability window, but
	// fast enough to reach a 50ms deadline. Parity is immediate, so a waitReady
	// that returned at first parity would report success here.
	clock := time.Now()
	e := &Engine{Backend: b, now: func() time.Time {
		clock = clock.Add(time.Millisecond)
		return clock
	}}

	err := e.waitReady(context.Background(), "rel", 50*time.Millisecond)
	require.Error(t, err)
	require.Contains(t, err.Error(), "timed out")
	require.Greater(t, b.calls, 1, "should keep polling rather than returning at first parity")
}

func TestWaitReadyReturnsOnceTheWindowElapses(t *testing.T) {
	prev := waitPollInterval
	waitPollInterval = time.Millisecond
	defer func() { waitPollInterval = prev }()

	young := []ServiceState{{Name: "api", Running: 1, Desired: 1, NewestTaskAge: time.Second}}
	aged := []ServiceState{{Name: "api", Running: 1, Desired: 1, NewestTaskAge: time.Minute}}

	b := &scriptedBackend{script: [][]ServiceState{young, young, aged}}
	e := &Engine{Backend: b, now: time.Now}

	require.NoError(t, e.waitReady(context.Background(), "rel", time.Hour))
	require.Equal(t, 3, b.calls, "must wait for the task to outlive the window, not return at parity")
}

// A replacement task after a lost rollout is newer, so the window it has left
// grows back and waitReady must keep waiting rather than treating regained
// parity as already-stable.
func TestWaitReadyWaitsOutAReplacementTasksFreshWindow(t *testing.T) {
	prev := waitPollInterval
	waitPollInterval = time.Millisecond
	defer func() { waitPollInterval = prev }()

	young := []ServiceState{{Name: "api", Running: 1, Desired: 1, NewestTaskAge: time.Second}}
	lost := []ServiceState{{Name: "api", Running: 0, Desired: 1}}
	aged := []ServiceState{{Name: "api", Running: 1, Desired: 1, NewestTaskAge: time.Minute}}

	b := &scriptedBackend{script: [][]ServiceState{young, lost, young, aged}}
	e := &Engine{Backend: b, now: time.Now}

	require.NoError(t, e.waitReady(context.Background(), "rel", time.Hour))
	require.Equal(t, 4, b.calls, "the fresh task's remaining window must still be waited out")
}

// A wedged rollout is reported immediately rather than after the full timeout.
func TestWaitReadyFailsFastOnWedgedRollout(t *testing.T) {
	prev := waitPollInterval
	waitPollInterval = time.Millisecond
	defer func() { waitPollInterval = prev }()

	b := &convergenceBackend{states: []ServiceState{{Name: "api", Running: 0, Desired: 1, UpdateState: "paused"}}}
	e := &Engine{Backend: b, now: time.Now}

	err := e.waitReady(context.Background(), "rel", time.Hour)
	require.Error(t, err)
	require.Contains(t, err.Error(), "update paused")
	require.Equal(t, 1, b.calls, "should fail on the first observation, not wait out the timeout")
}

// A cancelled wait ends at once, with the context's own error. The timeout it
// would otherwise serve out defaults to five minutes and is legitimately set
// higher, so a caller with a shutdown grace period measured in seconds cannot
// wait for it — and the release may be one that never converges at all.
func TestWaitReadyReturnsOnACancelledContext(t *testing.T) {
	// Long enough that nothing but the cancellation can end the sleep, and a
	// timeout to match, so the wait has exactly one way out.
	prev := waitPollInterval
	waitPollInterval = time.Hour
	defer func() { waitPollInterval = prev }()

	b := &convergenceBackend{states: []ServiceState{{Name: "api", Running: 0, Desired: 1}}}
	e := &Engine{Backend: b, now: time.Now}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan error, 1)
	go func() { done <- e.waitReady(ctx, "rel", time.Hour) }()

	select {
	case err := <-done:
		require.ErrorIs(t, err, context.Canceled, "the wait must report the cancellation, not a timeout")
	case <-time.After(5 * time.Second):
		t.Fatal("waitReady ignored the cancelled context and sat on its poll interval")
	}
}

// convergenceBackend is a Backend that only answers StackServices; waitReady
// touches nothing else.
type convergenceBackend struct {
	Backend
	states []ServiceState
	calls  int
}

func (b *convergenceBackend) StackServices(context.Context, string) []ServiceState {
	b.calls++
	return b.states
}

// scriptedBackend returns a different state per poll, then repeats the last.
type scriptedBackend struct {
	Backend
	script [][]ServiceState
	calls  int
}

func (b *scriptedBackend) StackServices(context.Context, string) []ServiceState {
	i := b.calls
	b.calls++
	if i >= len(b.script) {
		i = len(b.script) - 1
	}
	return b.script[i]
}
