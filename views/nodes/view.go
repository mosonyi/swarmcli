package nodesview

import (
	"fmt"
	"sort"
	"strings"
	"swarmcli/docker"
	"swarmcli/ui"
	filterlist "swarmcli/ui/components/filterable/list"
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
	// Compute proportional column widths (6 equal partitions) so header aligns with items
	labels := []string{"HOSTNAME", "ROLE", "STATE", "MANAGER", "ADDRESS", "LABELS"}
	width := m.List.Viewport.Width
	if width <= 0 {
		if m.width > 0 {
			width = m.width
		} else {
			width = 80
		}
	}
	cols := 6
	starts := make([]int, cols)
	for i := 0; i < cols; i++ {
		starts[i] = (i * width) / cols
	}
	colWidths := make([]int, cols)
	for i := 0; i < cols; i++ {
		if i == cols-1 {
			colWidths[i] = width - starts[i]
		} else {
			colWidths[i] = starts[i+1] - starts[i]
		}
		if colWidths[i] < 1 {
			colWidths[i] = 1
		}
	}
	// Prefix first label with a leading space to match item alignment
	labels[0] = " " + labels[0]
	header := ui.RenderColumnHeader(labels, colWidths)

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

	// Compute frame width/height
	frameWidth := m.List.Viewport.Width + 4
	// Reserve two lines from the viewport height for surrounding UI
	frameHeight := m.List.Viewport.Height - 2
	if frameHeight <= 0 {
		if m.height > 0 {
			frameHeight = m.height - 4
		}
		if frameHeight <= 0 {
			frameHeight = 20
		}
	}
	// Determine how many inner content lines will be shown and render exactly
	// that many lines without mutating the viewport height to avoid jitter.
	headerLines := 0
	if header != "" {
		headerLines = 1
	}
	footerLines := 0
	if footer != "" {
		footerLines = len(strings.Split(footer, "\n"))
	}
	desiredContentLines := frameHeight - 2 - headerLines - footerLines
	if desiredContentLines < 1 {
		desiredContentLines = 1
	}

	content := m.List.VisibleContent(desiredContentLines)

	return ui.RenderFramedBoxHeight(title, header, content, footer, frameWidth, frameHeight)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
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
