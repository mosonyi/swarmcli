// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package searchinput

import "github.com/charmbracelet/lipgloss"

// View renders the search input line. Returns "" when inactive.
func (m *Model) View() string {
	if !m.active {
		return ""
	}

	style := lipgloss.NewStyle().
		Background(lipgloss.Color("#303030")).
		Foreground(lipgloss.Color("#00d7ff")).
		Padding(0, 1)

	return style.Render("/ " + m.input.Value())
}
