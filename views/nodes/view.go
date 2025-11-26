package nodesview

import (
	"fmt"
	"strings"
	"swarmcli/docker"
	"swarmcli/ui"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}

	total := len(m.entries)
	managers := 0
	for _, n := range m.entries {
		if n.Manager {
			managers++
		}
	}

	title := fmt.Sprintf("Nodes (%d total, %d manager%s)", total, managers, plural(managers))
	content := m.renderNodes()
	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}

	// Blueish styled header
	header := renderHeader(m.entries)

	return ui.RenderFramedBox(title, header, content, "", width)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// renderHeader calculates column widths based on longest visible values.
func renderHeader(entries []docker.NodeEntry) string {
	if len(entries) == 0 {
		return "HOSTNAME  ROLE  STATE  MANAGER  ADDRESS"
	}

	colWidths := calcColumnWidths(entries)
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("75")). // blueish tone
		Bold(true)

	return headerStyle.Render(fmt.Sprintf(
		"%-*s  %-*s  %-*s  %-*s  %-*s",
		colWidths["Hostname"], "HOSTNAME",
		colWidths["Role"], "ROLE",
		colWidths["State"], "STATE",
		colWidths["Manager"], "MANAGER",
		colWidths["Addr"], "ADDRESS",
	))
}

// renderNodes builds the visible list of nodes with colorized header and cursor highlight.
func (m *Model) renderNodes() string {
	if len(m.entries) == 0 {
		return "No swarm nodes found."
	}

	colWidths := calcColumnWidths(m.entries)
	var lines []string

	for i, n := range m.entries {
		manager := "no"
		if n.Manager {
			manager = "yes"
		}

		line := fmt.Sprintf(
			"%-*s  %-*s  %-*s  %-*s  %-*s",
			colWidths["Hostname"], n.Hostname,
			colWidths["Role"], n.Role,
			colWidths["State"], n.State,
			colWidths["Manager"], manager,
			colWidths["Addr"], n.Addr,
		)

		if i == m.cursor {
			line = ui.CursorStyle.Render(line)
		}

		lines = append(lines, line)
	}

	status := fmt.Sprintf(" Node %d of %d ", m.cursor+1, len(m.entries))
	lines = append(lines, "", ui.StatusBarStyle.Render(status))

	return strings.Join(lines, "\n")
}

// calcColumnWidths determines the best width per column based on the longest cell.
func calcColumnWidths(entries []docker.NodeEntry) map[string]int {
	widths := map[string]int{
		"Hostname": len("HOSTNAME"),
		"Role":     len("ROLE"),
		"State":    len("STATE"),
		"Manager":  len("MANAGER"),
		"Addr":     len("ADDRESS"),
	}

	for _, e := range entries {
		if len(e.Hostname) > widths["Hostname"] {
			widths["Hostname"] = len(e.Hostname)
		}
		if len(e.Role) > widths["Role"] {
			widths["Role"] = len(e.Role)
		}
		if len(e.State) > widths["State"] {
			widths["State"] = len(e.State)
		}
		manager := "no"
		if e.Manager {
			manager = "yes"
		}
		if len(manager) > widths["Manager"] {
			widths["Manager"] = len(manager)
		}
		if len(e.Addr) > widths["Addr"] {
			widths["Addr"] = len(e.Addr)
		}
	}

	return widths
}
