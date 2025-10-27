package commandinput

import "github.com/charmbracelet/lipgloss"

func (m Model) View() string {
	if !m.visible {
		return ""
	}

	inputStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#303030")).
		Foreground(lipgloss.Color("#00d7ff")).
		Padding(0, 1)

	view := inputStyle.Render(m.input.View())

	if m.errorMsg != "" {
		errorStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff5f87")).
			Bold(true).
			Padding(0, 1)
		view += "\n" + errorStyle.Render(m.errorMsg)
	}

	return view
}
