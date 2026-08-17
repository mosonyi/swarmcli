// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

// countChains runs a command — flattening tea.Batch, whose members the runtime
// runs rather than the caller — and counts the ticks it schedules. Each
// scheduled tick is one live poll chain, because the loop re-arms from the
// poll's result.
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
		if _, isSpinner := msg.(SpinnerTickMsg); isSpinner {
			return // not part of the poll loop
		}
		run(m.Update(msg), depth+1)
	}
	run(cmd, 0)
	return chains
}

// firstTick runs a command and returns the tick it scheduled, so a test can
// hold on to one chain's tick while a later entry arms another.
func firstTick(t *testing.T, cmd tea.Cmd) TickMsg {
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

// Entering the view must start exactly one poll chain.
//
// switchToView batches the factory's command and OnEnter, and this view's
// factory returns Init(), so the two together are the whole of entry. A view
// arming in both ran two chains for its life, polling the swarm at twice the
// intended rate.
func TestEnteringTheViewArmsExactlyOneChain(t *testing.T) {
	fastPoll(t)
	m := testModel()

	require.Equal(t, 1, countChains(m, tea.Batch(m.Init(), m.OnEnter())),
		"the factory and OnEnter must not each start a chain")
}

// Returning to the view must restart polling.
//
// A chain does not survive a navigation: every view declares its own TickMsg
// type, so a tick belonging to a view that is no longer current is delivered to
// a different view and dropped. That tick is also the only thing that clears
// tickScheduled, so a guard the entry hook does not reset leaves the view
// permanently stale — armTick declines forever and nothing ever polls again.
func TestReturningToTheViewRestartsPolling(t *testing.T) {
	fastPoll(t)
	m := testModel()

	m.OnEnter()
	// The chain armed on the first entry is dropped mid-flight, exactly as a
	// drill-down drops it: the tick never comes back to clear the flag.
	require.True(t, m.tickScheduled)

	require.Equal(t, 1, countChains(m, m.OnEnter()),
		"goBack calls OnEnter and nothing else, so it has to restart the chain")
}

// A tick armed before a drill-down must not sustain a chain after the return.
//
// The chain usually dies on the way out, but one still in flight when the
// operator returns finds this view current again, and OnEnter has already armed
// a replacement — re-arming from both leaves the view polling at twice the rate
// for the rest of its life.
func TestStaleTickFromAnEarlierEntryIsDropped(t *testing.T) {
	fastPoll(t)
	m := testModel()

	stale := firstTick(t, m.OnEnter()) // the chain armed on the first entry
	m.OnEnter()                        // goBack: a fresh chain

	require.Equal(t, 0, countChains(m, m.Update(stale)),
		"a tick from an earlier entry must not arm a successor")
}
