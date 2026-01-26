// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package commandinput

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if !m.active {
		return ""
	}

	inputStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#303030")).
		Foreground(lipgloss.Color("#00d7ff")).
		Padding(0, 1)

	suggestionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080"))

	selectedStyle := suggestionStyle.
		Foreground(lipgloss.Color("#00d7ff")).
		Bold(true)

	// Build inline suggestion: if there's a selected suggestion and the
	// user has typed a prefix, show the prefix bold+green and the remainder
	// in blue appended to the input line.
	var suggestionLines []string
	inline := ""
	if len(m.suggestions) > 0 {
		sel := m.suggestions[m.selected]
		typed := strings.TrimSpace(m.input.Value())
		if typed != "" && strings.HasPrefix(sel, typed) {
			// matched prefix: keep the typed prefix rendered by the input
			// (so it uses the current input color/style) and append only the
			// remainder in blue as an inline suggestion with no extra space.
			suffixStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("12"))
			if len(sel) > len(typed) {
				inline = suffixStyle.Render(sel[len(typed):])
			}
		}
		// populate suggestion list beneath as before
		for i, s := range m.suggestions {
			if i == m.selected {
				suggestionLines = append(suggestionLines, selectedStyle.Render("> "+s))
			} else {
				suggestionLines = append(suggestionLines, suggestionStyle.Render("  "+s))
			}
		}
	}

	// Render input line with inline suggestion appended.
	// Use the raw Value() instead of input.View() so we can avoid the
	// cursor glyph rendered by the textinput helper and keep only a thin
	// caret (or none) visually.
	inputLine := "> " + m.input.Value() + inline

	return lipgloss.JoinVertical(
		lipgloss.Left,
		inputStyle.Render(inputLine),
		strings.Join(suggestionLines, "\n"),
	)
}
