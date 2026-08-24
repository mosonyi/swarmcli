// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

//go:build integration

package charts

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Eldara-Tech/swarmcli/v2/charts"
	"github.com/Eldara-Tech/swarmcli/v2/docker"
	swarmlog "github.com/Eldara-Tech/swarmcli/v2/utils/log"
)

const chartTemplate = `version: "3.9"

services:
  whoami:
    image: traefik/whoami:v1.10
    deploy:
      replicas: {{ .Values.replicas }}
      labels:
        com.swarmcli.release: {{ .Release.Name }}
`

// writeDemoChart writes a minimal chart into a temp dir and returns its path.
func writeDemoChart(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Chart.yaml"),
		[]byte("apiVersion: v1\nname: itest\nversion: 0.1.0\nappVersion: \"1.0\"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "values.yaml"),
		[]byte("replicas: 1\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "templates"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "templates", "stack.yaml"),
		[]byte(chartTemplate), 0o644))
	return dir
}

func TestChartsReleaseLifecycle(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()

	ctx := context.Background()
	release := fmt.Sprintf("itest-%d", time.Now().UnixNano())

	ch, err := charts.LoadChartDir(writeDemoChart(t))
	require.NoError(t, err)

	values, err := charts.MergeValues(ch.Values, nil, []string{"replicas=2"})
	require.NoError(t, err)

	manifest, err := charts.Render(ch, charts.RenderContext{
		Values:  values,
		Release: charts.ReleaseMeta{Name: release, Namespace: release, Revision: 1},
		Chart:   charts.ChartMeta{Name: ch.Metadata.Name, Version: ch.Metadata.Version},
	})
	require.NoError(t, err)

	eng := charts.NewEngine()

	// Ensure teardown even if assertions fail mid-way.
	defer func() { _, _ = eng.Uninstall(ctx, release, true) }()

	rel, err := eng.Install(ctx, release, charts.ReleaseChart{Name: ch.Metadata.Name, Version: ch.Metadata.Version},
		values, manifest, charts.InstallOptions{Wait: true, Timeout: 90 * time.Second})
	require.NoError(t, err, "install should succeed")
	require.Equal(t, 1, rel.Revision)
	require.Equal(t, charts.StatusDeployed, rel.Status)

	// The release-history Config must exist with the expected labels.
	cfg, err := docker.InspectConfig(ctx, fmt.Sprintf("swarmcli.release.%s.v1", release))
	require.NoError(t, err)
	require.Equal(t, charts.TypeRelease, cfg.Config.Spec.Labels[charts.LabelType])
	require.Equal(t, release, cfg.Config.Spec.Labels[charts.LabelRelease])
	require.Equal(t, "1", cfg.Config.Spec.Labels[charts.LabelRevision])

	// List should include the release.
	list, err := eng.List(ctx)
	require.NoError(t, err)
	found := false
	for _, r := range list {
		if r.Name == release {
			found = true
		}
	}
	require.True(t, found, "release should appear in list")

	// Status should report the deployed revision and at least one service.
	cur, svcs, err := eng.Status(ctx, release)
	require.NoError(t, err)
	require.Equal(t, charts.StatusDeployed, cur.Status)
	require.NotEmpty(t, svcs, "status should list services")

	// Upgrade to revision 2 (scale to 1) then roll back to revision 1's content.
	up, err := charts.MergeValues(ch.Values, nil, []string{"replicas=1"})
	require.NoError(t, err)
	upManifest, err := charts.Render(ch, charts.RenderContext{
		Values:  up,
		Release: charts.ReleaseMeta{Name: release, Namespace: release, Revision: 2},
		Chart:   charts.ChartMeta{Name: ch.Metadata.Name, Version: ch.Metadata.Version},
	})
	require.NoError(t, err)
	rel2, err := eng.Upgrade(ctx, release, charts.ReleaseChart{Name: ch.Metadata.Name, Version: ch.Metadata.Version},
		up, upManifest, charts.InstallOptions{Wait: true, Timeout: 90 * time.Second})
	require.NoError(t, err)
	require.Equal(t, 2, rel2.Revision)

	hist, err := eng.History(ctx, release)
	require.NoError(t, err)
	require.Len(t, hist, 2)

	rb, err := eng.Rollback(ctx, release, 1, charts.InstallOptions{Wait: true, Timeout: 90 * time.Second})
	require.NoError(t, err)
	require.Equal(t, 3, rb.Revision)

	// Uninstall removes the stack and the release Configs.
	_, err = eng.Uninstall(ctx, release, true)
	require.NoError(t, err)
	_, err = docker.InspectConfig(ctx, fmt.Sprintf("swarmcli.release.%s.v1", release))
	require.Error(t, err, "release config should be gone after uninstall")
}

// packChartTgz packages dir into a gzipped tar whose entries are nested under
// prefix/, matching the layout LoadChartArchive expects.
func packChartTgz(t *testing.T, dir, prefix string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	err := filepath.Walk(dir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		body, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		hdr := &tar.Header{Name: prefix + "/" + filepath.ToSlash(rel), Mode: 0o644, Size: int64(len(body)), Typeflag: tar.TypeReg}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		_, err = tw.Write(body)
		return err
	})
	require.NoError(t, err)
	require.NoError(t, tw.Close())
	require.NoError(t, gz.Close())
	return buf.Bytes()
}

// TestChartsRepoInstallLifecycle exercises the full repository path end to end:
// a chart served over HTTP is added, searched, resolved, pulled, rendered, and
// installed against the live Swarm, then uninstalled. This covers the
// add→search→resolve→pull chain that the engine-only lifecycle test skips.
func TestChartsRepoInstallLifecycle(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()

	ctx := context.Background()

	// Serve an index.yaml plus the packaged chart from a local HTTP server.
	const tgzName = "itest-0.1.0.tgz"
	tgz := packChartTgz(t, writeDemoChart(t), "itest")
	index := "apiVersion: v1\n" +
		"entries:\n" +
		"  itest:\n" +
		"    - name: itest\n" +
		"      version: \"0.1.0\"\n" +
		"      description: integration test chart\n" +
		"      urls: [\"" + tgzName + "\"]\n"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasSuffix(r.URL.Path, "/index.yaml"):
			_, _ = w.Write([]byte(index))
		case strings.HasSuffix(r.URL.Path, tgzName):
			_, _ = w.Write(tgz)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	// Add the repo (downloads + validates the index), then find and fetch the
	// chart through the public store API.
	store := charts.NewRepoStoreAt(t.TempDir())
	store.AllowPlaintext = true // the httptest server above serves plain http
	require.NoError(t, store.Add("itest-repo", srv.URL))

	hits, err := store.Search("itest")
	require.NoError(t, err)
	require.NotEmpty(t, hits, "search should find the served chart")

	entry, base, err := store.Resolve("itest-repo/itest", "")
	require.NoError(t, err)

	ch, err := store.Pull(entry, base)
	require.NoError(t, err)
	require.Equal(t, "itest", ch.Metadata.Name)

	// Render the pulled chart and install it on the live Swarm.
	release := fmt.Sprintf("itest-repo-%d", time.Now().UnixNano())
	values, err := charts.MergeValues(ch.Values, nil, []string{"replicas=2"})
	require.NoError(t, err)
	manifest, err := charts.Render(ch, charts.RenderContext{
		Values:  values,
		Release: charts.ReleaseMeta{Name: release, Namespace: release, Revision: 1},
		Chart:   charts.ChartMeta{Name: ch.Metadata.Name, Version: ch.Metadata.Version},
	})
	require.NoError(t, err)

	eng := charts.NewEngine()
	defer func() { _, _ = eng.Uninstall(ctx, release, true) }()

	rel, err := eng.Install(ctx, release, charts.ReleaseChart{Name: ch.Metadata.Name, Version: ch.Metadata.Version},
		values, manifest, charts.InstallOptions{Wait: true, Timeout: 90 * time.Second})
	require.NoError(t, err, "install of pulled chart should succeed")
	require.Equal(t, charts.StatusDeployed, rel.Status)

	_, svcs, err := eng.Status(ctx, release)
	require.NoError(t, err)
	require.NotEmpty(t, svcs, "status should list services")

	_, err = eng.Uninstall(ctx, release, true)
	require.NoError(t, err)
	_, err = docker.InspectConfig(ctx, fmt.Sprintf("swarmcli.release.%s.v1", release))
	require.Error(t, err, "release config should be gone after uninstall")
}

// writeExtNetChart writes a chart whose stack attaches to (and declares as
// external) the given network name, returning the chart dir. The network is
// declared in requirements.yaml with autoCreate:true so the pre-flight creates
// it.
func writeExtNetChart(t *testing.T, netName string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Chart.yaml"),
		[]byte("apiVersion: v1\nname: itest\nversion: 0.1.0\nappVersion: \"1.0\"\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "values.yaml"),
		[]byte("replicas: 1\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "requirements.yaml"),
		[]byte("networks:\n  - name: "+netName+"\n    autoCreate: true\n    description: itest overlay\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "templates"), 0o755))
	stack := "version: \"3.9\"\n\n" +
		"services:\n" +
		"  whoami:\n" +
		"    image: traefik/whoami:v1.10\n" +
		"    networks:\n      - " + netName + "\n" +
		"    deploy:\n      replicas: {{ .Values.replicas }}\n\n" +
		"networks:\n  " + netName + ":\n    external: true\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "templates", "stack.yaml"), []byte(stack), 0o644))
	return dir
}

// TestChartsExternalNetworkAutoCreate verifies that installing a chart which
// declares a not-yet-existing external network auto-creates it as a swarm
// overlay and then deploys successfully (rather than failing the deploy).
func TestChartsExternalNetworkAutoCreate(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()

	ctx := context.Background()
	netName := fmt.Sprintf("itest-ext-%d", time.Now().UnixNano())
	release := fmt.Sprintf("itest-extnet-%d", time.Now().UnixNano())

	ch, err := charts.LoadChartDir(writeExtNetChart(t, netName))
	require.NoError(t, err)
	values, err := charts.MergeValues(ch.Values, nil, nil)
	require.NoError(t, err)
	manifest, err := charts.Render(ch, charts.RenderContext{
		Values:  values,
		Release: charts.ReleaseMeta{Name: release, Namespace: release, Revision: 1},
		Chart:   charts.ChartMeta{Name: ch.Metadata.Name, Version: ch.Metadata.Version},
	})
	require.NoError(t, err)

	eng := charts.NewEngine()
	defer func() {
		_, _ = eng.Uninstall(ctx, release, true)
		_ = docker.RemoveNetwork(ctx, netName) // external nets survive stack rm
	}()

	// Precondition: the external network does not exist yet.
	nets, err := docker.ListNetworks(ctx)
	require.NoError(t, err)
	for _, n := range nets {
		require.NotEqual(t, netName, n.Name, "test network must not pre-exist")
	}

	rel, err := eng.Install(ctx, release, charts.ReleaseChart{Name: ch.Metadata.Name, Version: ch.Metadata.Version},
		values, manifest, charts.InstallOptions{Wait: true, Timeout: 90 * time.Second, Requirements: ch.Requirements})
	require.NoError(t, err, "install should auto-create the external network and deploy")
	require.Equal(t, charts.StatusDeployed, rel.Status)
	require.Equal(t, []string{netName}, rel.ManagedNetworks, "the auto-created network is recorded on the revision")

	// The external network now exists and is swarm-scoped.
	nets, err = docker.ListNetworks(ctx)
	require.NoError(t, err)
	found := false
	for _, n := range nets {
		if n.Name == netName {
			found = true
			require.Equal(t, "swarm", n.Scope, "auto-created network should be swarm-scoped")
		}
	}
	require.True(t, found, "external network should have been auto-created")

	// Uninstall leaves the auto-created network in place and reports it as
	// orphaned so the operator can reclaim it.
	res, err := eng.Uninstall(ctx, release, true)
	require.NoError(t, err)
	require.Equal(t, []string{netName}, res.OrphanedNetworks)
	nets, err = docker.ListNetworks(ctx)
	require.NoError(t, err)
	stillThere := false
	for _, n := range nets {
		if n.Name == netName {
			stillThere = true
		}
	}
	require.True(t, stillThere, "uninstall must not remove the shared external network")
}
