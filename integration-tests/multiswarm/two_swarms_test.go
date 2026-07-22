// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

//go:build integration

// Package multiswarm proves that an Engine built with charts.NewDockerBackend
// addresses exactly the swarm it names, and no other.
//
// This needs TWO swarms to mean anything. The failure mode #469/#487 exist to
// prevent is silent: a backend that deploys to one swarm while reading release
// history, networks and convergence from another reports success either way, so
// a single-swarm test passes whether or not the seam works. Two contexts
// pointing at the SAME daemon would be equally worthless.
//
// Swarm A is the shared multi-node swarm the rest of the integration suite runs
// against (context "swarmcli"). Swarm B is a throwaway single-node dind started
// here, following the pattern in integration-tests/swarmlock.
package multiswarm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Eldara-Tech/swarmcli/charts"
	swarmlog "github.com/Eldara-Tech/swarmcli/utils/log"
)

const (
	dindImage     = "docker:29-dind"
	containerName = "swarmcli-multiswarm-dind"
	dindHostPort  = "24375" // distinct from the shared swarm (22375) and swarmlock (23375)
	contextB      = "swarmcli-multiswarm-b"
	hostB         = "tcp://localhost:" + dindHostPort

	// contextA is the shared swarm's context, created by test-setup/testenv.sh.
	contextA = "swarmcli"
)

const chartTemplate = `version: "3.9"

services:
  whoami:
    image: traefik/whoami:v1.10
    deploy:
      replicas: 1
`

func hostDocker(t *testing.T, args ...string) (string, error) {
	t.Helper()
	out, err := exec.Command("docker", append([]string{"--context", "default"}, args...)...).CombinedOutput()
	return string(out), err
}

func innerB(t *testing.T, args ...string) (string, error) {
	t.Helper()
	return hostDocker(t, append([]string{"exec", containerName, "docker"}, args...)...)
}

// startSwarmB brings up the throwaway second swarm and registers a context for
// it, returning once its daemon answers and it is a swarm manager.
func startSwarmB(t *testing.T) {
	t.Helper()

	_, _ = hostDocker(t, "rm", "-f", containerName)
	_ = exec.Command("docker", "context", "rm", "-f", contextB).Run()
	t.Cleanup(func() {
		_ = exec.Command("docker", "context", "rm", "-f", contextB).Run()
		_, _ = hostDocker(t, "rm", "-f", containerName)
	})

	out, err := hostDocker(t, "run", "-d", "--privileged",
		"-e", "DOCKER_TLS_CERTDIR=",
		"-p", dindHostPort+":2375",
		"--name", containerName, dindImage)
	require.NoErrorf(t, err, "starting swarm B dind: %s", out)

	deadline := time.Now().Add(90 * time.Second)
	for {
		if _, err := innerB(t, "version"); err == nil {
			break
		}
		require.Truef(t, time.Now().Before(deadline), "swarm B daemon never became ready")
		time.Sleep(time.Second)
	}

	out, err = innerB(t, "swarm", "init", "--advertise-addr", "eth0")
	require.NoErrorf(t, err, "swarm init on B: %s", out)

	out, err = hostDocker(t, "context", "create", contextB, "--docker", "host="+hostB)
	require.NoErrorf(t, err, "creating context B: %s", out)
}

func demoChart(t *testing.T) *charts.Chart {
	t.Helper()
	dir := t.TempDir()
	write := func(name, body string) {
		t.Helper()
		require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dir, name)), 0o755))
		require.NoError(t, os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644))
	}
	write("Chart.yaml", "apiVersion: v1\nname: multiswarm\nversion: 0.1.0\nappVersion: \"1.0\"\n")
	write("values.yaml", "{}\n")
	write("templates/stack.yaml", chartTemplate)

	ch, err := charts.LoadChartDir(dir)
	require.NoError(t, err)
	return ch
}

func engineFor(ctxName string) *charts.Engine {
	return charts.NewEngineWith(charts.NewDockerBackend(ctxName))
}

// A release installed through a backend naming one swarm must exist on that
// swarm and be invisible from the other — in BOTH directions. One direction
// alone would pass against a backend that always read swarm A.
func TestReleasesDoNotLeakBetweenSwarms(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()
	startSwarmB(t)

	ctx := context.Background()
	ch := demoChart(t)

	onA := fmt.Sprintf("multiswarm-a-%d", time.Now().UnixNano())
	onB := fmt.Sprintf("multiswarm-b-%d", time.Now().UnixNano())

	engA, engB := engineFor(contextA), engineFor(contextB)

	manifest, err := charts.Render(ch, charts.RenderContext{
		Values:  map[string]any{},
		Release: charts.ReleaseMeta{Name: onA, Namespace: onA, Revision: 1},
		Chart:   charts.ChartMeta{Name: "multiswarm", Version: "0.1.0"},
	})
	require.NoError(t, err)

	_, err = engA.Install(ctx, onA, charts.ReleaseChart{Name: "multiswarm", Version: "0.1.0"},
		map[string]any{}, manifest, charts.InstallOptions{})
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = engA.Uninstall(context.Background(), onA, false) })

	manifestB, err := charts.Render(ch, charts.RenderContext{
		Values:  map[string]any{},
		Release: charts.ReleaseMeta{Name: onB, Namespace: onB, Revision: 1},
		Chart:   charts.ChartMeta{Name: "multiswarm", Version: "0.1.0"},
	})
	require.NoError(t, err)

	_, err = engB.Install(ctx, onB, charts.ReleaseChart{Name: "multiswarm", Version: "0.1.0"},
		map[string]any{}, manifestB, charts.InstallOptions{})
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = engB.Uninstall(context.Background(), onB, false) })

	namesOn := func(e *charts.Engine) []string {
		rels, err := e.List(ctx)
		require.NoError(t, err)
		var out []string
		for _, r := range rels {
			out = append(out, r.Name)
		}
		return out
	}

	// Release history lives in Docker Configs read through the SDK client, which
	// is the half #469's proposed fix would have left pointing at the ambient
	// swarm.
	require.Contains(t, namesOn(engA), onA, "A's own release must be visible from A")
	require.NotContains(t, namesOn(engA), onB, "B's release must NOT be visible from A")

	require.Contains(t, namesOn(engB), onB, "B's own release must be visible from B")
	require.NotContains(t, namesOn(engB), onA, "A's release must NOT be visible from B")
}

// The deploy half goes through `docker --context <name> stack deploy`, a
// different mechanism from the SDK reads above. Assert the stack really landed
// on the named daemon by asking that daemon directly, not through swarmcli.
func TestDeployLandsOnTheNamedSwarm(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()
	startSwarmB(t)

	ctx := context.Background()
	ch := demoChart(t)
	release := fmt.Sprintf("multiswarm-land-%d", time.Now().UnixNano())

	manifest, err := charts.Render(ch, charts.RenderContext{
		Values:  map[string]any{},
		Release: charts.ReleaseMeta{Name: release, Namespace: release, Revision: 1},
		Chart:   charts.ChartMeta{Name: "multiswarm", Version: "0.1.0"},
	})
	require.NoError(t, err)

	engB := engineFor(contextB)
	_, err = engB.Install(ctx, release, charts.ReleaseChart{Name: "multiswarm", Version: "0.1.0"},
		map[string]any{}, manifest, charts.InstallOptions{})
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = engB.Uninstall(context.Background(), release, false) })

	// Ask swarm B's own daemon, bypassing swarmcli entirely.
	out, err := innerB(t, "stack", "ls", "--format", "{{.Name}}")
	require.NoErrorf(t, err, "listing stacks on B: %s", out)
	require.Contains(t, out, release, "the stack must exist on the swarm the backend named")

	// And it must not have landed on the shared swarm.
	outA, err := exec.Command("docker", "--context", contextA, "stack", "ls", "--format", "{{.Name}}").CombinedOutput()
	require.NoErrorf(t, err, "listing stacks on A: %s", outA)
	require.NotContains(t, string(outA), release, "the stack must NOT have been deployed to the other swarm")
}
