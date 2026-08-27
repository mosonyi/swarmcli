// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"time"
)

type tickMsg time.Time

// tickInterval is a variable so a test can arm the timer without waiting for
// it: invoking a tea.Tick command runs its sleep synchronously.
var tickInterval = 5 * time.Second

func tick() tea.Cmd {
	return tea.Tick(tickInterval, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// handleTick drives the app's periodic work and re-arms itself.
//
// The re-arm was missing, so the timer armed in Init fired once and stopped:
// the header's context name and its container and service counts froze a few
// seconds after launch and never moved again. The drift check needs a timer
// that keeps running, and the header was always meant to have one.
func (m *Model) handleTick(msg tickMsg) (tea.Model, tea.Cmd) {
	return m, tea.Batch(
		tick(),
		m.systemInfo.LoadStatus(),
		checkContextDriftCmd(),
	)
}
