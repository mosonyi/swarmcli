// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package commandinput

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if !m.active {
		return nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			val := strings.TrimSpace(m.input.Value())
			m.Hide()
			return func() tea.Msg { return SubmitMsg{Command: val} }

		case "esc":
			m.Hide()
			return nil

		case "up":
			if len(m.suggestions) > 0 {
				m.selected = (m.selected - 1 + len(m.suggestions)) % len(m.suggestions)
			}
		case "down":
			if len(m.suggestions) > 0 {
				m.selected = (m.selected + 1) % len(m.suggestions)
			}
		case "tab":
			if len(m.suggestions) > 0 {
				m.input.SetValue(m.suggestions[m.selected] + " ")
				m.input.CursorEnd()
				m.refreshSuggestions()
			}
		default:
			// Update suggestions when typing
			m.input, cmd = m.input.Update(msg)
			m.refreshSuggestions()
			return cmd
		}
	}

	m.input, cmd = m.input.Update(msg)
	return cmd
}
