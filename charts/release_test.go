// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// fakeBackend is an in-memory Backend for lifecycle tests.
type fakeBackend struct {
	configs       map[string]fakeConfig
	deployed      map[string]string // stack name -> manifest
	volumes       map[string][]string
	services      map[string][]ServiceState
	networkScopes map[string]string // network name -> scope
	createNetErr  map[string]error  // network name -> error to return on create
	failNext      bool
	rmStackErr    error
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
func (f *fakeBackend) CreateConfig(_ context.Context, name string, data []byte, labels map[string]string) error {
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
