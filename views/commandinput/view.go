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

	var suggestionLines []string
	for i, s := range m.suggestions {
		if i == m.selected {
			suggestionLines = append(suggestionLines, selectedStyle.Render("> "+s))
		} else {
			suggestionLines = append(suggestionLines, suggestionStyle.Render("  "+s))
		}
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		inputStyle.Render(m.input.View()),
		strings.Join(suggestionLines, "\n"),
	)
}
