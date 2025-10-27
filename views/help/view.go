package helpview

import (
	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	title := lipgloss.NewStyle().
		Bold(true).
		Background(lipgloss.Color("57")).
		Foreground(lipgloss.Color("230")).
		Padding(0, 1).
		Render(" HELP — press q to return ")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		m.viewport.View(),
	)

	return content
}
