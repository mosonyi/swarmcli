package helpview

import (
	"fmt"
	"swarmcli/ui"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}

	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00d7ff")).
		Render("Available Commands")

	body := m.content
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Render("[press q or esc to go back]")

	return ui.BorderStyle.Render(
		fmt.Sprintf("%s\n\n%s\n\n%s", header, body, footer),
	)
}
