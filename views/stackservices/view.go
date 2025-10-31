package stackservicesview

import (
	"fmt"
	"strings"
	"swarmcli/ui"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if !m.Visible {
		return ""
	}

	title := fmt.Sprintf("Services on Node (Total: %d)", len(m.entries))
	header := "SERVICE                        STACK                REPLICAS"

	content := m.viewport.View()
	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}

	return ui.RenderFramedBox(title, header, content, width)
}

func (m Model) renderEntries() string {
	if len(m.entries) == 0 {
		return "No services found for this node."
	}

	var lines []string

	// Header row
	header := lipgloss.NewStyle().
		Bold(true).
		Underline(true).
		Render(fmt.Sprintf("%-30s %-20s %-10s", "SERVICE", "STACK", "REPLICAS"))
	lines = append(lines, header)

	for i, e := range m.entries {
		replicas := fmt.Sprintf("%d/%d", e.ReplicasOnNode, e.ReplicasTotal)

		// Colorize replica count
		switch {
		case e.ReplicasTotal == 0:
			replicas = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("—")
		case e.ReplicasOnNode == 0:
			replicas = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(replicas) // red
		case e.ReplicasOnNode < e.ReplicasTotal:
			replicas = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(replicas) // yellow
		default:
			replicas = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(replicas) // green
		}

		line := fmt.Sprintf("%-30s %-20s %-10s", e.ServiceName, e.StackName, replicas)

		if i == m.cursor {
			line = ui.CursorStyle.Render(line)
		}

		lines = append(lines, line)
	}

	// Footer with cursor info
	status := fmt.Sprintf(" Service %d of %d ", m.cursor+1, len(m.entries))
	lines = append(lines, "", ui.StatusBarStyle.Render(status))

	return strings.Join(lines, "\n")
}
