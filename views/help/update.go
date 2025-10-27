package helpview

import (
	"strings"
	"swarmcli/commands"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m Model) Update(msg tea.Msg) (view.View, tea.Cmd) {
	return m, nil
}

func (m Model) buildContent() string {
	cmds := commands.All()
	var sb strings.Builder

	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")).
		Bold(true).
		Render("Available Commands\n\n")
	sb.WriteString(header)

	for _, cmd := range cmds {
		name := lipgloss.NewStyle().
			Foreground(lipgloss.Color("212")).
			Render(cmd.Name())
		desc := lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Render(cmd.Description())
		sb.WriteString(name + " - " + desc + "\n")
	}
	sb.WriteString("\nPress q to return.")

	return sb.String()
}
