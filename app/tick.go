// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"time"
)

type tickMsg time.Time

func tick() tea.Cmd {
	return tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func (m *Model) handleTick(msg tickMsg) (tea.Model, tea.Cmd) {
	return m, m.systemInfo.LoadStatus()
}
