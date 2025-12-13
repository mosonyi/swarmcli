package nodesview

import (
	"fmt"
	"sort"
	"strings"
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
	header := renderHeader(m.colWidths)

	// Footer: cursor + optional search query
	status := fmt.Sprintf("Node %d of %d", m.List.Cursor+1, len(m.List.Filtered))
	statusBar := ui.StatusBarStyle.Render(status)

	var footer string
	if m.List.Mode == filterlist.ModeSearching {
		footer = ui.StatusBarStyle.Render("Filter (type then Enter): " + m.List.Query)
	} else if m.List.Query != "" {
		footer = ui.StatusBarStyle.Render("Filter: " + m.List.Query)
	}

	if footer != "" {
		footer = statusBar + "\n" + footer
	} else {
		footer = statusBar
	}

	content := m.List.View()

	// Add 4 to make frame full terminal width (app reduces viewport by 4 in normal mode)
	frameWidth := m.List.Viewport.Width + 4
	headerLines := 0
	if header != "" {
		headerLines = 1
	}
	footerLines := 0
	if footer != "" {
		footerLines = len(strings.Split(footer, "\n"))
	}
	frameHeight := m.List.Viewport.Height + 2 + headerLines + footerLines
	if frameHeight <= 0 {
		frameHeight = 20
	}
	return ui.RenderFramedBoxHeight(title, header, content, footer, frameWidth, frameHeight)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// renderHeader uses pre-calculated column widths.
func renderHeader(colWidths map[string]int) string {
	if len(colWidths) == 0 {
		return "HOSTNAME  ROLE  STATE  MANAGER  ADDRESS"
	}

	headerStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")). // white
		Bold(true)

	return headerStyle.Render(fmt.Sprintf(
		"%-*s        %-*s        %-*s        %-*s        %-*s        %-*s",
		colWidths["Hostname"], "HOSTNAME",
		colWidths["Role"], "ROLE",
		colWidths["State"], "STATE",
		colWidths["Manager"], "MANAGER",
		colWidths["Addr"], "ADDRESS",
		colWidths["Labels"], "LABELS",
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
		"Labels":   len("LABELS"),
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
		// Format labels as key=value pairs
		labelsStr := formatLabels(e.Labels)
		if len(labelsStr) > widths["Labels"] {
			widths["Labels"] = len(labelsStr)
		}
	}

	return widths
}

// formatLabels converts label map to comma-separated key=value string
func formatLabels(labels map[string]string) string {
	if len(labels) == 0 {
		return "-"
	}

	var parts []string
	for k, v := range labels {
		parts = append(parts, fmt.Sprintf("%s=%s", k, v))
	}
	// Sort for consistent display
	sort.Strings(parts)
	return strings.Join(parts, ",")
}
