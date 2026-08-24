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

	"github.com/Eldara-Tech/swarmcli/v2/charts"
	swarmlog "github.com/Eldara-Tech/swarmcli/v2/utils/log"
)

// waveChart writes a chart whose single service is the stack given, so a
// release file can put a fast one and a never-converging one in different waves.
func waveChart(t *testing.T, name, stack string) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Chart.yaml"),
		[]byte("apiVersion: v1\nname: "+name+"\nversion: 0.1.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "values.yaml"), []byte("{}\n"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "templates"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "templates", "stack.yaml"), []byte(stack), 0o644))
	return dir
}

// fastStack comes up as soon as swarm can schedule it.
//
// Two choices here are about making "fast" true rather than hoped for, and both
// were learned the hard way: the first version of this used alpine on any node
// and the healthy wave took longer than the whole apply's timeout, failing on the
// release that was supposed to work.
//
//   - The same image the rest of this package deploys, so whichever node runs it
//     has already pulled it. A cold pull is most of what a first deploy spends.
//   - Pinned to the manager, so it lands on the node the suite has been using
//     rather than on a worker seeing the image for the first time.
const fastStack = `version: "3.9"

services:
  app:
    image: traefik/whoami:v1.10
    deploy:
      replicas: 1
      placement:
        constraints: [node.role == manager]
`

// waveReleaseFile writes a release file from name/wave/chart triples.
func waveReleaseFile(t *testing.T, entries ...waveEntry) *charts.ReleaseFile {
	t.Helper()
	body := "apiVersion: v1\nreleases:\n"
	for _, e := range entries {
		body += fmt.Sprintf("  - name: %s\n    chart: %s\n    wave: %d\n", e.release, e.chart, e.wave)
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "swarmcli-release.yaml")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))

	rf, err := charts.LoadReleaseFile(path)
	require.NoError(t, err)
	return rf
}

type waveEntry struct {
	release string
	wave    int
	chart   string
}

// Releases apply in wave order regardless of the order the file lists them.
//
// The unit suite proves the grouping against a fake; only a real swarm proves
// that the thing the grouping waits on — swarm's own account of whether a stack
// has converged — actually gates the next wave.
func TestApplyRespectsWaves(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()

	ctx := context.Background()
	stamp := time.Now().UnixNano()
	first := fmt.Sprintf("itest-wave-a-%d", stamp)
	second := fmt.Sprintf("itest-wave-b-%d", stamp)
	third := fmt.Sprintf("itest-wave-c-%d", stamp)

	chart := waveChart(t, "wavey", fastStack)
	// Written last-wave-first on purpose: an apply that ignored waves and used
	// file order would deploy them in exactly the wrong sequence, so this cannot
	// pass by accident.
	rf := waveReleaseFile(t,
		waveEntry{third, 2, chart},
		waveEntry{first, 0, chart},
		waveEntry{second, 1, chart},
	)

	eng := charts.NewEngine()
	src := charts.NewChartSource(nil)
	defer func() {
		for _, r := range []string{first, second, third} {
			_, _ = eng.Uninstall(ctx, r, true)
		}
	}()

	plan, err := eng.PlanApply(ctx, rf, src, charts.PlanOptions{})
	require.NoError(t, err)
	require.Equal(t, []string{first, second, third},
		[]string{plan.Releases[0].Name, plan.Releases[1].Name, plan.Releases[2].Name},
		"the plan is in wave order, not the file's")

	results, err := eng.Apply(ctx, plan, charts.InstallOptions{Timeout: 120 * time.Second})
	require.NoError(t, err)
	require.Equal(t, []string{first, second, third},
		[]string{results[0].Name, results[1].Name, results[2].Name})

	// Every wave really did converge before the next started, which is what the
	// revision timestamps record. Each is written by the deploy that created it,
	// and the barrier sits between them.
	for _, r := range []string{first, second, third} {
		revs, err := eng.History(ctx, r)
		require.NoError(t, err)
		require.Len(t, revs, 1, "release %s should have been installed exactly once", r)
	}
}

// A wave that does not converge stops every wave after it.
//
// This is the acceptance criterion for the whole feature, and the reason it is
// here rather than only in the unit suite: "did not converge" is swarm's
// judgement, not ours. The second wave is a task no node can schedule, so it
// stays pending — nothing is pulled and nothing is started — and the third wave
// must show no service and, more tellingly, no revision record at all.
func TestApplyStopsAtAWaveThatDoesNotConverge(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()

	ctx := context.Background()
	stamp := time.Now().UnixNano()
	first := fmt.Sprintf("itest-wavefail-a-%d", stamp)
	stuck := fmt.Sprintf("itest-wavefail-b-%d", stamp)
	never := fmt.Sprintf("itest-wavefail-c-%d", stamp)

	rf := waveReleaseFile(t,
		waveEntry{first, 0, waveChart(t, "ok", fastStack)},
		waveEntry{stuck, 1, waveChart(t, "stuck", unschedulableStack)},
		waveEntry{never, 2, waveChart(t, "never", fastStack)},
	)

	eng := charts.NewEngine()
	src := charts.NewChartSource(nil)
	defer func() {
		for _, r := range []string{first, stuck, never} {
			_, _ = eng.Uninstall(ctx, r, true)
		}
	}()

	plan, err := eng.PlanApply(ctx, rf, src, charts.PlanOptions{})
	require.NoError(t, err)

	// The timeout bounds *each* wave, not the apply, so it has to be comfortable
	// for the healthy wave that goes first — and everything above it is the cost
	// of the wave that never converges. Both halves of that trade are real, and
	// it is the same number an operator has to pick for `--timeout`.
	const timeout = 45 * time.Second
	start := time.Now()
	results, err := eng.Apply(ctx, plan, charts.InstallOptions{Timeout: timeout})
	elapsed := time.Since(start)

	require.Error(t, err, "a wave that never converges must fail the apply")
	require.Contains(t, err.Error(), "timed out")
	require.Contains(t, err.Error(), stuck, "the release that held the wave up is the one to name")
	require.GreaterOrEqual(t, elapsed, timeout, "the barrier must actually be waited out")

	require.Equal(t, []string{first, stuck},
		[]string{results[0].Name, results[1].Name},
		"a partial apply reports what it did")
	require.Len(t, results, 2, "nothing in wave 2 was applied")

	// The strongest statement available: wave 2 has no stored revision, so Apply
	// never reached it. An empty service list alone would be weaker — that is
	// also what a deploy that was accepted and then failed looks like.
	_, err = eng.History(ctx, never)
	require.Error(t, err, "wave 2 must have no release record at all")

	// And the contrast that makes it mean something: wave 1 does have one. It
	// was deployed and only then failed to converge, because deployAndRecord
	// records the revision before it waits.
	revs, err := eng.History(ctx, stuck)
	require.NoError(t, err)
	require.Len(t, revs, 1)
}
