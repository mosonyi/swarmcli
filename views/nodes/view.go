package nodesview

import (
	"fmt"
	"strings"
	"swarmcli/ui"

	"github.com/charmbracelet/lipgloss"
)

var (
	cursorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("63")).
		Bold(true)

	statusBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		Background(lipgloss.Color("237")).
		Padding(0, 1)
)

func (m Model) View() string {
	if !m.Visible {
		return ""
	}

	total := len(m.nodes)
	managers := 0
	for _, n := range m.nodes {
		if n.ManagerStatus != "" {
			managers++
		}
	}

	title := fmt.Sprintf("Nodes (%d total, %d manager%s)", total, managers, plural(managers))
	header := "HOSTNAME              STATUS     AVAILABILITY   MANAGER STATUS"

	content := m.viewport.View()
	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}

	return ui.RenderFramedBox(title, header, content, width)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// renderNodes builds the visible list of nodes with colorized header and cursor highlight.
func (m Model) renderNodes() string {
	if len(m.nodes) == 0 {
		return "No swarm nodes found."
	}

	var lines []string

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
