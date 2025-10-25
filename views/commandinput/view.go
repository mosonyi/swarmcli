package commandinput

import "github.com/charmbracelet/lipgloss"

// View renders the command bar and optional error message.
func (m Model) View() string {
	if !m.visible {
		return ""
	}

	cmdBarStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#303030")).
		Foreground(lipgloss.Color("#00d7ff")).
		Padding(0, 1)

	view := cmdBarStyle.Render(m.input.View())

	if m.errorMsg != "" {
		errStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff5f87")).
			Bold(true).
			Padding(0, 1)
		view += "\n" + errStyle.Render(m.errorMsg)
	}

	return view
}
