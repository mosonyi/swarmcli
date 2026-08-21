// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

//go:build integration

package charts

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/Eldara-Tech/swarmcli/v2/charts"
	swarmlog "github.com/Eldara-Tech/swarmcli/v2/utils/log"
)

// slowStartDelay is how long the fixture below takes to report healthy. It has
// to be comfortably longer than the poll interval and the stability window so
// an early return is unmistakable rather than a timing coincidence.
const slowStartDelay = 15 * time.Second

// slowStartStack is a service that is slow to become READY while being quick to
// be SCHEDULED — which is precisely the gap #473 fixed and the existing suite
// cannot express.
//
// A Swarm task carrying a healthcheck stays in task state `starting` until the
// first healthy event: swarmkit's container controller blocks in Start() waiting
// for one before the agent reports the task running. So gating the check on a
// file the container creates only after a delay keeps the task out of `running`
// for exactly that long. alpine is already used by test-setup/test-stack.yml, so
// this needs no extra pull and no network.
//
// start_period MUST cover the delay. Outside it, `retries` consecutive failures
// mark the container unhealthy, and swarmkit answers that by shutting the
// container down — the task would restart-loop instead of coming up slowly, and
// the test would assert the wrong thing. Inside start_period a failing check
// costs nothing, while a passing one still promotes to healthy immediately.
const slowStartStack = `version: "3.9"

services:
  slow:
    image: alpine:latest
    command: ["sh", "-c", "sleep %d; touch /tmp/ready; while true; do sleep 3600; done"]
    healthcheck:
      test: ["CMD-SHELL", "test -f /tmp/ready"]
      interval: 2s
      retries: 3
      start_period: %s
    deploy:
      replicas: 1
`

// slowStartGrace is start_period: comfortably longer than slowStartDelay so a
// slow CI runner cannot turn "not ready yet" into "unhealthy, kill it".
const slowStartGrace = 3 * slowStartDelay

// unschedulableStack can never converge: no node carries the label, so the task
// stays `pending` forever. Nothing is ever pulled or started.
const unschedulableStack = `version: "3.9"

services:
  nowhere:
    image: alpine:latest
    command: ["sh", "-c", "sleep 3600"]
    deploy:
      replicas: 1
      placement:
        constraints:
          - node.labels.swarmcli_absent_label == true
`

// --wait must not return until the tasks are genuinely running.
//
// This is the regression the rest of the suite cannot express. Every existing
// test passes Wait:true and asserts only that the call eventually succeeds,
// which was equally true before #473 — back then it returned as soon as swarm
// had SCHEDULED the tasks. Timing the call against a service that is fast to
// schedule and slow to become healthy is what tells the two apart.
func TestWaitBlocksUntilTasksAreActuallyRunning(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()

	ctx := context.Background()
	release := fmt.Sprintf("itest-wait-%d", time.Now().UnixNano())
	manifest := fmt.Sprintf(slowStartStack, int(slowStartDelay.Seconds()), slowStartGrace)

	eng := charts.NewEngine()
	defer func() { _, _ = eng.Uninstall(ctx, release, true) }()

	start := time.Now()
	_, err := eng.Install(ctx, release, charts.ReleaseChart{Name: "itest", Version: "0.1.0"},
		nil, manifest, charts.InstallOptions{Wait: true, Timeout: 120 * time.Second})
	elapsed := time.Since(start)

	require.NoError(t, err, "the service does become healthy, so --wait must succeed")
	require.GreaterOrEqual(t, elapsed, slowStartDelay,
		"--wait returned after %s, before the task could possibly be running (%s): it is reporting scheduled, not running", elapsed, slowStartDelay)
}

// A service that can never converge must hit the timeout, not report success.
//
// The positive case alone would pass for a --wait that blocked forever, so this
// is what makes the pair a real regression test rather than half of one.
func TestWaitTimesOutOnAServiceThatCanNeverConverge(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()

	ctx := context.Background()
	release := fmt.Sprintf("itest-stuck-%d", time.Now().UnixNano())
	const timeout = 20 * time.Second

	eng := charts.NewEngine()
	defer func() { _, _ = eng.Uninstall(ctx, release, true) }()

	start := time.Now()
	_, err := eng.Install(ctx, release, charts.ReleaseChart{Name: "itest", Version: "0.1.0"},
		nil, unschedulableStack, charts.InstallOptions{Wait: true, Timeout: timeout})
	elapsed := time.Since(start)

	require.Error(t, err, "a task pinned to a label no node carries never runs; --wait must not report success")
	require.Contains(t, err.Error(), "timed out")
	require.GreaterOrEqual(t, elapsed, timeout, "the timeout must actually be waited out")
}
