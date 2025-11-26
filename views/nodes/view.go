package nodesview

import (
	"fmt"
	"swarmcli/docker"
	"swarmcli/ui"
	filterlist "swarmcli/ui/components/filterable/list"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}

	total := len(m.List.Items)
	managers := 0
	for _, n := range m.List.Items {
		if n.Manager {
			managers++
		}
	}

	title := fmt.Sprintf("Nodes (%d total, %d manager%s)", total, managers, plural(managers))
	header := renderHeader(m.List.Items)

	// Footer: cursor + optional search query
	status := fmt.Sprintf("Node %d of %d", m.List.Cursor+1, len(m.List.Filtered))
	statusBar := ui.StatusBarStyle.Render(status)

	var footer string
	if m.List.Mode == filterlist.ModeSearching {
		footer = ui.StatusBarStyle.Render("Filter: " + m.List.Query)
	}

	if footer != "" {
		footer = statusBar + "\n" + footer
	} else {
		footer = statusBar
	}

	content := m.List.View()

	return ui.RenderFramedBox(title, header, content, footer, m.List.Viewport.Width)
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
