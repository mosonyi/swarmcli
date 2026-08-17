// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package nodesview

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

	// The real factory, not a hand-written stand-in for it: switchToView
	// batches the factory's command and OnEnter, and standing in for the
	// factory with m.Init() is how one view's second arm survived the first
	// pass at this.
	v, loadCmd := factory(m.deps, 80, 24, nil)
	entered := v.(*Model)
	require.Equal(t, 1, countChains(entered, tea.Batch(entered.Init(), loadCmd, entered.OnEnter())),
		"the factory and OnEnter must not each start a chain")
}

// A tick keeps the chain alive on its own, and produces exactly one successor.
func TestTickSustainsExactlyOneChain(t *testing.T) {
	fastTick(t)
	m := testModel()
	m.Visible = true

	require.Equal(t, 1, countChains(m, m.Update(TickMsg{Gen: m.pollGen})),
		"a tick must schedule exactly one successor")

	// The poll reporting no change must not schedule another on top of it.
	require.Equal(t, 0, countChains(m, m.Update(PollRetryMsg{})),
		"PollRetryMsg must not re-arm; the tick already did")
}

// A chain does not survive a navigation: every view declares its own TickMsg
// type, so a tick belonging to a view that is no longer current is delivered to
// a different view and dropped. Returning must therefore re-arm, or the view
// comes back permanently stale.
func TestReturningToTheViewRestartsPolling(t *testing.T) {
	fastTick(t)
	m := testModel()

	require.Equal(t, 1, countChains(m, m.OnEnter()),
		"goBack calls OnEnter and nothing else, so it has to restart the chain")
}

// firstTick runs a command and returns the tick it scheduled, so a test can
// hold on to one chain's tick while a later entry arms another.
func firstTick(t *testing.T, m *Model, cmd tea.Cmd) TickMsg {
	t.Helper()
	var found *TickMsg
	var run func(c tea.Cmd)
	run = func(c tea.Cmd) {
		if c == nil || found != nil {
			return
		}
		msg := c()
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, sub := range batch {
				run(sub)
			}
			return
		}
		if tick, ok := msg.(TickMsg); ok {
			found = &tick
		}
	}
	run(cmd)
	require.NotNil(t, found, "the command scheduled no tick")
	return *found
}

// A tick armed before a drill-down must not sustain a chain after the return.
//
// The chain usually dies on the way out: its tick is delivered to whichever
// view is current by then, and dropped. But one still in flight when the
// operator returns finds this view current again, and OnEnter has already
// armed a replacement — re-arming from both leaves the view polling at twice
// the rate for the rest of its life.
func TestStaleTickFromAnEarlierEntryIsDropped(t *testing.T) {
	fastTick(t)
	m := testModel()

	stale := firstTick(t, m, m.OnEnter()) // the chain armed on the first entry
	m.OnEnter()                           // goBack: a fresh chain

	require.Equal(t, 0, countChains(m, m.Update(stale)),
		"a tick from an earlier entry must not arm a successor")
}
