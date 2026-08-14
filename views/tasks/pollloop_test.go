// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package tasksview

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

// countChains runs a command — flattening tea.Batch, whose members the runtime
// runs rather than the caller — and counts the ticks it schedules. Each
// scheduled tick is one live poll chain, because a tick re-arms itself.
//
// Messages other than ticks are fed back into Update exactly once, so a load
// that arms a tick of its own is counted too. That is the shape of the bug:
// the arm was not in one place.
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
		run(m.Update(msg), depth+1)
	}
	run(cmd, 0)
	return chains
}

func fastTick(t *testing.T) {
	t.Helper()
	original := PollInterval
	PollInterval = time.Millisecond
	t.Cleanup(func() { PollInterval = original })
}

// Entering a view must start exactly one poll chain.
//
// switchToView batches the factory's command AND OnEnter, so a view that armed
// in both — or in a load handler reached from either — ran two chains for the
// life of the view, polling the daemon at twice the intended rate.
func TestEnteringTheViewArmsExactlyOneChain(t *testing.T) {
	fastTick(t)
	m := testModel()

	// What switchToView does: the factory's command, then OnEnter.
	entry := tea.Batch(m.Init(), LoadTasksCmd(m.stackName), m.OnEnter())
	require.Equal(t, 1, countChains(m, entry),
		"the factory and OnEnter must not each start a chain")
}

// A tick keeps the chain alive on its own, and produces exactly one successor.
func TestTickSustainsExactlyOneChain(t *testing.T) {
	fastTick(t)
	m := testModel()
	m.visible = true

	require.Equal(t, 1, countChains(m, m.Update(TickMsg(time.Now()))),
		"a tick must schedule exactly one successor")

	// The poll reporting no change must not schedule another on top of it.
	require.Equal(t, 0, countChains(m, m.Update(PollRetryMsg{})),
		"PollRetryMsg must not re-arm; the tick already did")
}

// A chain cannot survive a navigation: every view declares its own TickMsg
// type, so a tick belonging to a view that is no longer current is delivered to
// a different view and dropped. Returning must therefore re-arm, or the view
// comes back permanently stale.
func TestReturningToTheViewRestartsPolling(t *testing.T) {
	fastTick(t)
	m := testModel()

	require.Equal(t, 1, countChains(m, m.OnEnter()),
		"goBack calls OnEnter and nothing else, so it has to restart the chain")
}
