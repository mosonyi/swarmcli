// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package searchinput

import tea "github.com/charmbracelet/bubbletea"

// Update handles key messages while the search input is active.
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if !m.active {
		return nil
	}

	// Passive mode: ignore all input (app handles / and Esc at its level).
	if !m.editing {
		return nil
	}

	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.Type {
		case tea.KeyEnter:
			// Lock the box into passive mode; query stays visible.
			m.Confirm()
			return nil

		case tea.KeyEsc:
			m.Hide()
			return func() tea.Msg { return SearchClearedMsg{} }

		case tea.KeyBackspace:
			if m.input.Value() == "" {
				// Backspace on empty input dismisses and clears.
				m.Hide()
				return func() tea.Msg { return SearchClearedMsg{} }
			}
			m.input, cmd = m.input.Update(msg)
			q := m.input.Value()
			return tea.Batch(cmd, func() tea.Msg { return SearchQueryMsg{Query: q} })

		default:
			m.input, cmd = m.input.Update(msg)
			q := m.input.Value()
			return tea.Batch(cmd, func() tea.Msg { return SearchQueryMsg{Query: q} })
		}
	}

	m.input, cmd = m.input.Update(msg)
	return cmd
}
