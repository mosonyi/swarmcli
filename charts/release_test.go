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
	networkScopes map[string]string   // network name -> scope
	secrets       map[string]struct{} // existing secret names
	createNetErr  map[string]error    // network name -> error to return on create
	failNext      bool
	rmStackErr    error
	refreshErr    error
	secretsErr    error                   // error to return from SecretNames
	onCreate      func(name string) error // hook to simulate concurrent config creation
	deleteCfgErr  map[string]error        // config name -> error to return on delete
}

type fakeConfig struct {
	data   []byte
	labels map[string]string
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

func (f *fakeBackend) DeployStack(name, manifest string) error {
	if f.failNext {
		f.failNext = false
		return fmt.Errorf("boom")
	}
	f.deployed[name] = manifest
	return nil
}
func (f *fakeBackend) RemoveStack(name string) error {
	if f.rmStackErr != nil {
		return f.rmStackErr
	}
	delete(f.deployed, name)
	return nil
}
func (f *fakeBackend) RefreshSnapshot() error { return f.refreshErr }
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
		out = append(out, ConfigMeta{Name: name, Labels: c.labels})
	}
	return out, nil
}
func (f *fakeBackend) InspectConfig(_ context.Context, name string) ([]byte, error) {
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
func (f *fakeBackend) StackServices(name string) []ServiceState { return f.services[name] }
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
func (f *fakeBackend) CreateOverlayNetwork(_ context.Context, name string) error {
	if err := f.createNetErr[name]; err != nil {
		return err
	}
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

func TestUninstallRemovesStackAndConfigsKeepsVolumes(t *testing.T) {
	fb := newFakeBackend()
	fb.volumes["demo"] = []string{"demo_data"}
	e := testEngine(fb)
	ctx := context.Background()
	_, err := e.Install(ctx, "demo", ReleaseChart{Name: "demo", Version: "1"}, nil, "services:\n  s:\n    image: x\n", InstallOptions{})
	require.NoError(t, err)

	require.NoError(t, e.Uninstall(ctx, "demo", false))
	require.Empty(t, fb.deployed)
	require.Empty(t, fb.configs)
	require.Equal(t, []string{"demo_data"}, fb.volumes["demo"]) // volume retained

	require.Error(t, e.Uninstall(ctx, "demo", false)) // already gone
}

func TestUninstallPurgeVolumes(t *testing.T) {
	fb := newFakeBackend()
	fb.volumes["demo"] = []string{"demo_data"}
	e := testEngine(fb)
	ctx := context.Background()
	_, _ = e.Install(ctx, "demo", ReleaseChart{Name: "demo", Version: "1"}, nil, "services:\n  s:\n    image: x\n", InstallOptions{})
	require.NoError(t, e.Uninstall(ctx, "demo", true))
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

// A converged service whose update is still in flight must not count as ready.
func TestAllConvergedRejectsInProgress(t *testing.T) {
	require.True(t, allConverged([]ServiceState{{Replicas: "2/2", Status: "active"}}))
	require.True(t, allConverged([]ServiceState{{Replicas: "1/1", Status: "updated"}}))
	require.False(t, allConverged([]ServiceState{{Replicas: "2/2", Status: "updating"}}))
	require.False(t, allConverged([]ServiceState{{Replicas: "1/1", Status: "rolling back"}}))
	require.False(t, allConverged([]ServiceState{{Replicas: "1/2", Status: "active"}}))
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
	err = e.Uninstall(ctx, "demo", false)
	require.ErrorContains(t, err, "removing stack")
	require.Empty(t, fb.configs) // history cleaned up despite the stack-removal failure
}
