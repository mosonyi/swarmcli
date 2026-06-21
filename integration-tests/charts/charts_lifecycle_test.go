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
	"swarmcli/docker"
	swarmlog "swarmcli/utils/log"
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
		[]byte("apiVersion: v2\nname: itest\nversion: 0.1.0\nappVersion: \"1.0\"\n"), 0o644))
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
	defer func() { _ = eng.Uninstall(ctx, release, true) }()

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

	// Uninstall removes the stack and the release Configs.
	require.NoError(t, eng.Uninstall(ctx, release, true))
	_, err = docker.InspectConfig(ctx, fmt.Sprintf("swarmcli.release.%s.v1", release))
	require.Error(t, err, "release config should be gone after uninstall")
}
