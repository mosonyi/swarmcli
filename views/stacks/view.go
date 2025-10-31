package stacksview

import (
	"fmt"
	"strings"
	"swarmcli/docker"
	"swarmcli/ui"

	"github.com/charmbracelet/lipgloss"
)

func (m Model) View() string {
	if !m.Visible {
		return ""
	}

	total := len(m.entries)
	title := fmt.Sprintf("Stacks (%d total)", total)
	content := m.renderStacks()
	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}

	header := renderHeader(m.entries)
	return ui.RenderFramedBox(title, header, content, width)
}

// --- HEADER ---

func renderHeader(entries []docker.StackEntry) string {
	if len(entries) == 0 {
		return "STACK  SERVICES  NODE COUNT"
	}

	colWidths := calcColumnWidths(entries)
	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("75")). // blueish tone
		Bold(true)

	return headerStyle.Render(fmt.Sprintf(
		"%-*s  %-*s  %-*s",
		colWidths["StackName"], "STACK",
		colWidths["ServiceCount"], "SERVICES",
		colWidths["NodeCount"], "NODE COUNT",
	))
}

// --- RENDER STACKS LIST ---

func (m Model) renderStacks() string {
	if len(m.entries) == 0 {
		return "No stacks found."
	}

	colWidths := calcColumnWidths(m.entries)
	var lines []string

	for i, s := range m.entries {
		line := fmt.Sprintf(
			"%-*s  %-*d  %-*d",
			colWidths["StackName"], s.Name,
			colWidths["ServiceCount"], s.ServiceCount,
			colWidths["NodeCount"], s.NodeCount,
		)

		if i == m.cursor {
			line = ui.CursorStyle.Render(line)
		}
		lines = append(lines, line)
	}

	status := fmt.Sprintf(" Stack %d of %d ", m.cursor+1, len(m.entries))
	lines = append(lines, "", ui.StatusBarStyle.Render(status))
	return strings.Join(lines, "\n")
}

// --- COLUMN WIDTHS ---

func calcColumnWidths(entries []docker.StackEntry) map[string]int {
	widths := map[string]int{
		"StackName":    len("STACK"),
		"ServiceCount": len("SERVICES"),
		"NodeCount":    len("NODE COUNT"),
	}

	for _, e := range entries {
		if len(e.Name) > widths["StackName"] {
			widths["StackName"] = len(e.Name)
		}

		if l := len(fmt.Sprintf("%d", e.ServiceCount)); l > widths["ServiceCount"] {
			widths["ServiceCount"] = l
		}

		if l := len(fmt.Sprintf("%d", e.NodeCount)); l > widths["NodeCount"] {
			widths["NodeCount"] = l
		}
	}

	return widths
}
