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

// writeExtConfigChart writes a chart whose service mounts a config declared
// external under `key`, optionally with a sibling `name:` naming the real
// resource. A nil name yields the historical key-as-name form.
func writeExtConfigChart(t *testing.T, key, name string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Chart.yaml"),
		[]byte("apiVersion: v1\nname: extname\nversion: 0.1.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "values.yaml"), []byte("{}\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "templates"), 0o755))

	decl := "configs:\n  " + key + ":\n    external: true\n"
	if name != "" {
		decl = "configs:\n  " + key + ":\n    external: true\n    name: " + name + "\n"
	}
	stack := "version: \"3.9\"\n\nservices:\n  app:\n" +
		"    image: traefik/whoami:v1.10\n" +
		"    configs:\n      - " + key + "\n" +
		"    deploy:\n      replicas: 1\n" +
		"      labels:\n        com.swarmcli.release: {{ .Release.Name }}\n\n" + decl
	require.NoError(t, os.WriteFile(filepath.Join(dir, "templates", "stack.yaml"), []byte(stack), 0o644))
	return dir
}

// A chart declaring an external config the way the Compose specification
// currently documents — `external: true` with a sibling `name:` — must install.
//
// This is the one thing the unit tests cannot prove. They check that swarmcli
// resolves the reference to the sibling name, against a fake backend. What
// makes the fix correct is that swarmcli's pre-flight and the *deploy* agree on
// which resource is meant, and the deploy is `docker stack deploy` shelled out
// to whatever `docker` binary is on PATH — not the docker/cli the module pins.
// Reading that loader's source establishes the intent; only a real swarm
// establishes that the binary in front of it behaves the same way.
//
// Before the fix the pre-flight refused this chart outright, naming the compose
// key and suggesting the operator create a second, unrelated config (#513).
func TestChartsExternalConfigResolvesASiblingName(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()

	ctx := context.Background()
	release := fmt.Sprintf("itest-extname-%d", time.Now().UnixNano())
	realName := release + "-real-config"

	_, err := docker.CreateConfig(ctx, realName, []byte("payload\n"), nil)
	require.NoError(t, err)

	eng := charts.NewEngine()
	defer func() {
		_, _ = eng.Uninstall(ctx, release, true)
		_ = docker.DeleteConfig(ctx, realName)
	}()

	// The compose key is deliberately NOT the name of anything on the swarm, so
	// resolving to the key cannot accidentally pass.
	chartDir := writeExtConfigChart(t, "alias", realName)
	ch, err := charts.LoadChartDir(chartDir)
	require.NoError(t, err)
	rctx := charts.RenderContext{
		Values:  ch.Values,
		Release: charts.ReleaseMeta{Name: release, Namespace: release, Revision: 1},
		Chart:   charts.ChartMeta{Name: ch.Metadata.Name, Version: ch.Metadata.Version},
	}
	manifest, err := charts.Render(ch, rctx)
	require.NoError(t, err)

	rel, err := eng.Install(ctx, release,
		charts.ReleaseChart{Name: ch.Metadata.Name, Version: ch.Metadata.Version},
		ch.Values, manifest, charts.InstallOptions{Wait: true, Timeout: 90 * time.Second})
	require.NoError(t, err, "a chart declaring an external config by name: must install")
	require.Equal(t, charts.StatusDeployed, rel.Status)
}

// The absent case, on a real swarm: the resource is reported and remediated by
// the name the chart asked for, never by the compose key. Getting this wrong is
// how #513 sent operators to create a second config that would not have helped.
func TestChartsExternalConfigMissingIsReportedByItsResolvedName(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()

	ctx := context.Background()
	release := fmt.Sprintf("itest-extmissing-%d", time.Now().UnixNano())
	absent := release + "-absent-config"

	eng := charts.NewEngine()
	defer func() { _, _ = eng.Uninstall(ctx, release, true) }()

	chartDir := writeExtConfigChart(t, "alias", absent)
	ch, err := charts.LoadChartDir(chartDir)
	require.NoError(t, err)
	manifest, err := charts.Render(ch, charts.RenderContext{
		Values:  ch.Values,
		Release: charts.ReleaseMeta{Name: release, Namespace: release, Revision: 1},
		Chart:   charts.ChartMeta{Name: ch.Metadata.Name, Version: ch.Metadata.Version},
	})
	require.NoError(t, err)

	_, err = eng.Install(ctx, release,
		charts.ReleaseChart{Name: ch.Metadata.Name, Version: ch.Metadata.Version},
		ch.Values, manifest, charts.InstallOptions{})
	require.Error(t, err)
	require.Contains(t, err.Error(), absent)
	require.Contains(t, err.Error(), "docker config create "+absent)
	require.NotContains(t, err.Error(), "alias", "the compose key is not the name of anything")
}

// The historical form still works, against the same real swarm. Every chart in
// Eldara-Tech/swarmcli-charts is written this way, so a regression here breaks
// every published chart at once.
func TestChartsExternalConfigStillResolvesTheMapKey(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()

	ctx := context.Background()
	release := fmt.Sprintf("itest-extkey-%d", time.Now().UnixNano())
	key := release + "-keyed-config"

	_, err := docker.CreateConfig(ctx, key, []byte("payload\n"), nil)
	require.NoError(t, err)

	eng := charts.NewEngine()
	defer func() {
		_, _ = eng.Uninstall(ctx, release, true)
		_ = docker.DeleteConfig(ctx, key)
	}()

	chartDir := writeExtConfigChart(t, key, "")
	ch, err := charts.LoadChartDir(chartDir)
	require.NoError(t, err)
	manifest, err := charts.Render(ch, charts.RenderContext{
		Values:  ch.Values,
		Release: charts.ReleaseMeta{Name: release, Namespace: release, Revision: 1},
		Chart:   charts.ChartMeta{Name: ch.Metadata.Name, Version: ch.Metadata.Version},
	})
	require.NoError(t, err)

	rel, err := eng.Install(ctx, release,
		charts.ReleaseChart{Name: ch.Metadata.Name, Version: ch.Metadata.Version},
		ch.Values, manifest, charts.InstallOptions{Wait: true, Timeout: 90 * time.Second})
	require.NoError(t, err, "the key-as-name form must keep working")
	require.Equal(t, charts.StatusDeployed, rel.Status)
}
