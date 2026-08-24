// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package contexts

import (
	"testing"
	"time"

	"github.com/Eldara-Tech/swarmcli/v2/docker"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

func fastTick(t *testing.T) {
	t.Helper()
	original := PollInterval
	PollInterval = time.Millisecond
	t.Cleanup(func() { PollInterval = original })
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
		if _, isTick := msg.(RefreshTickMsg); isTick {
			chains++
			return
		}
		run(m.Update(msg), depth+1)
	}
	run(cmd, 0)
	return chains
}

// firstTick runs a command and returns the tick it scheduled, so a test can
// hold on to one chain's tick while a later entry arms another.
func firstTick(t *testing.T, cmd tea.Cmd) RefreshTickMsg {
	t.Helper()
	var found *RefreshTickMsg
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
		if tick, ok := msg.(RefreshTickMsg); ok {
			found = &tick
		}
	}
	run(cmd)
	require.NotNil(t, found, "the command scheduled no tick")
	return *found
}

// Entering the view must start exactly one poll chain. switchToView batches the
// factory's command and OnEnter, so a view arming in both ran two chains for
// its whole life, re-reading the context store at twice the intended rate.
func TestEnteringTheViewArmsExactlyOneChain(t *testing.T) {
	fastTick(t)

	v, loadCmd := factory(docker.Deps{Contexts: noopContextOps()}, 80, 24, nil)
	entered := v.(*Model)
	entered.List.RenderItem = testModel().List.RenderItem
	require.Equal(t, 1, countChains(entered, tea.Batch(entered.Init(), loadCmd, entered.OnEnter())),
		"the factory and OnEnter must not each start a chain")
}

// Returning to the view must restart polling.
//
// A chain does not survive a navigation: every view declares its own tick type,
// so a tick belonging to a view that is no longer current is delivered to a
// different view and dropped. A factory that arms and an OnEnter that does not
// leaves the view permanently stale after the first drill-down, because goBack
// runs OnEnter and nothing else.
func TestReturningToTheViewRestartsPolling(t *testing.T) {
	fastTick(t)
	m := testModel()

	require.Equal(t, 1, countChains(m, m.OnEnter()))
}

// A tick armed before a drill-down must not sustain a chain after the return.
//
// The chain usually dies on the way out, but one still in flight when the
// operator returns finds this view current again, and OnEnter has already armed
// a replacement — re-arming from both leaves the view polling at twice the rate
// for the rest of its life.
func TestStaleTickFromAnEarlierEntryIsDropped(t *testing.T) {
	fastTick(t)
	m := testModel()

	stale := firstTick(t, m.OnEnter()) // the chain armed on the first entry
	m.OnEnter()                        // goBack: a fresh chain

	require.Equal(t, 0, countChains(m, m.Update(stale)),
		"a tick from an earlier entry must not arm a successor")
}
