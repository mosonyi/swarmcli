package nodesview

import (
	"fmt"
	"strings"
	"swarmcli/styles"

	"github.com/charmbracelet/lipgloss"
)

// View renders the nodes view.
func (m Model) View() string {
	if !m.Visible {
		return ""
	}

	header := "Nodes"

	content := fmt.Sprintf(
		"%s\n\n%s",
		header,
		m.viewport.View(),
	)

	return styles.BorderStyle.Render(content)
}

var (
	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("81")).
			Bold(true).
			Underline(true)

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("63")).
			Bold(true)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Background(lipgloss.Color("237")).
			Padding(0, 1)
)

// renderNodes builds the visible list of nodes with colorized header and cursor highlight.
func (m Model) renderNodes() string {
	if len(m.nodes) == 0 {
		return "No swarm nodes found."
	}

	var lines []string

	// Header
	header := fmt.Sprintf("%-20s %-10s %-12s %-15s", "HOSTNAME", "STATUS", "AVAILABILITY", "MANAGER STATUS")
	lines = append(lines, headerStyle.Render(header))
	lines = append(lines, strings.Repeat("─", len(header)))

	// Node rows
	for i, n := range m.nodes {
		line := fmt.Sprintf("%-20s %-10s %-12s %-15s", n.Hostname, n.Status, n.Availability, n.ManagerStatus)
		if i == m.cursor {
			line = cursorStyle.Render(line)
		}
		lines = append(lines, line)
	}

	// Footer / status bar
	status := fmt.Sprintf(" Node %d of %d ", m.cursor+1, len(m.nodes))
	lines = append(lines, "", statusBarStyle.Render(status))

	return strings.Join(lines, "\n")
}
