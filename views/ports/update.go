// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package portsview

import (
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		m.ready = true
		m.updateViewport()
		return nil

	case TickMsg:
		if m.ready {
			if m.deps.Snapshot != nil {
				m.deps.Snapshot.TriggerRefreshIfNeeded()
			}
			m.updateViewport()
		}
		return tickCmd()

	case tea.KeyMsg:
		switch msg.String() {
		case "esc":
			return func() tea.Msg {
				return view.GoBackMsg{}
			}
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return cmd
}
