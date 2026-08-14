// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package configsview

import (
	"context"
	"testing"
	"time"

	"github.com/Eldara-Tech/swarmcli/docker"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/require"
)

// drivePoll models the bubbletea runtime for one round: run each pending
// command, flattening tea.Batch — whose members the runtime runs, not the
// caller — feed every resulting message back into Update, and return the
// commands that came out.
//
// Flattening is the point. Asserting on the command Update returns cannot see
// past a batch, nor see a successor that arrives one message later, so a test
// written that way passes whatever the loop actually does.
func drivePoll(m *Model, pending []tea.Cmd) []tea.Cmd {
	var next []tea.Cmd
	var run func(c tea.Cmd)
	run = func(c tea.Cmd) {
		if c == nil {
			return
		}
		msg := c()
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, sub := range batch {
				run(sub)
			}
			return
		}
		if _, isSpinner := msg.(SpinnerTickMsg); isSpinner {
			return // not part of the poll loop
		}
		if cmd := m.Update(msg); cmd != nil {
			next = append(next, cmd)
		}
	}
	for _, c := range pending {
		run(c)
	}
	return next
}

// The poll loop must be a loop, not a tree.
//
// A tick that starts a poll and re-arms the ticker, while the poll's "nothing
// changed" result re-arms it too, yields two successors per beat — and each of
// those does the same. An idle view left open climbs into thousands of
// concurrent Docker reads within a minute.
func TestPollLoopDoesNotMultiply(t *testing.T) {
	original := PollInterval
	PollInterval = time.Millisecond
	t.Cleanup(func() { PollInterval = original })

	polled := false
	ops := noopConfigOps()
	inner := ops.listConfigsFn
	ops.listConfigsFn = func(ctx context.Context) ([]swarm.Config, error) {
		polled = true
		return inner(ctx)
	}
	m := testModel(func(m *Model) { m.deps = docker.Deps{Configs: ops} })

	// Steady state: the poll finds exactly what is already loaded (the ops
	// list nothing), so it reports no change. That is what a view left open
	// does all day.
	m.Update(configsLoadedMsg(nil))
	require.Equal(t, stateReady, m.state)

	pending := []tea.Cmd{tickCmd()}
	for round := 0; round < 12; round++ {
		pending = drivePoll(m, pending)
		require.LessOrEqual(t, len(pending), 1,
			"round %d: %d commands in flight — the loop has branched", round, len(pending))
		time.Sleep(2 * time.Millisecond)
	}
	require.True(t, polled, "the fixture must actually poll, or this proves nothing")
	require.Len(t, pending, 1, "and the loop must still be alive at the end")
}

// countChains runs a command — flattening tea.Batch, whose members the runtime
// runs rather than the caller — and counts the ticks it schedules. Each
// scheduled tick is one live poll chain, because a tick re-arms itself.
func countChains(m *Model, cmd tea.Cmd) int {
	chains := 0
	var run func(c tea.Cmd, depth int)
	run = func(c tea.Cmd, depth int) {
		if c == nil || depth > 2 {
			return
		}
		msg := c()
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, sub := range batch {
				run(sub, depth)
			}
			return
		}
		if _, isTick := msg.(TickMsg); isTick {
			chains++
			return
		}
		if _, isSpinner := msg.(SpinnerTickMsg); isSpinner {
			return
		}
		run(m.Update(msg), depth+1)
	}
	run(cmd, 0)
	return chains
}

// Entering a view must start exactly one poll chain. switchToView batches the
// factory's command AND OnEnter, so a view arming in both ran two chains for
// its whole life, polling the daemon at twice the intended rate.
func TestEnteringTheViewArmsExactlyOneChain(t *testing.T) {
	original := PollInterval
	PollInterval = time.Millisecond
	t.Cleanup(func() { PollInterval = original })

	m := testModel()
	require.Equal(t, 1, countChains(m, tea.Batch(m.Init(), m.OnEnter())),
		"the factory and OnEnter must not each start a chain")
}

// A chain cannot survive a navigation: every view declares its own TickMsg
// type, so a tick belonging to a view that is no longer current is delivered to
// a different view and dropped. Returning must therefore re-arm, or the view
// comes back permanently stale. goBack calls OnEnter and nothing else.
func TestReturningToTheViewRestartsPolling(t *testing.T) {
	original := PollInterval
	PollInterval = time.Millisecond
	t.Cleanup(func() { PollInterval = original })

	m := testModel()
	require.Equal(t, 1, countChains(m, m.OnEnter()))
}
