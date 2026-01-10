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
	// Compute proportional column widths (8 equal partitions) so header aligns with items
	labels := []string{"ID", "HOSTNAME", "ROLE", "STATE", "MANAGER", "VERSION", "ADDRESS", "LABELS"}
	width := m.List.Viewport.Width
	if width <= 0 {
		if m.width > 0 {
			width = m.width
		} else {
			width = 80
		}
	}
	cols := 8
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

	frame := ui.ComputeFrameDimensions(
		m.List.Viewport.Width,
		m.List.Viewport.Height,
		m.width,
		m.height,
		header,
		footer,
	)
	if frame.DesiredContentLines < 1 {
		frame.DesiredContentLines = 1
	}

	content := m.List.VisibleContent(frame.DesiredContentLines)

	framed := ui.RenderFramedBox(title, header, content, footer, frame.FrameWidth)

	if m.confirmDialog.Visible {
		framed = ui.OverlayCentered(framed, m.confirmDialog.View(), frame.FrameWidth, frame.FrameHeight)
	}

	return framed
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
		"ID":       len("ID"),
		"Hostname": len("HOSTNAME"),
		"Role":     len("ROLE"),
		"State":    len("STATE"),
		"Manager":  len("MANAGER"),
		"Version":  len("VERSION"),
		"Addr":     len("ADDRESS"),
		"Labels":   len("LABELS"),
	}

	for _, e := range entries {
		if len(e.ID) > widths["ID"] {
			widths["ID"] = len(e.ID)
		}
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
		if len(e.Version) > widths["Version"] {
			widths["Version"] = len(e.Version)
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

// formatLabelsWithScroll formats labels with horizontal scroll offset and truncation indicator
func formatLabelsWithScroll(labels map[string]string, offset int, maxWidth int) string {
	full := formatLabels(labels)
	if full == "-" {
		return full
	}

	// Apply scroll offset
	if offset > len(full) {
		offset = len(full)
	}
	visible := full[offset:]

	// Truncate if needed and add > indicator
	if len(visible) > maxWidth {
		if maxWidth > 1 {
			visible = visible[:maxWidth-1] + ">"
		} else {
			visible = ">"
		}
	}

	return visible
}
