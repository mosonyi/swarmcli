// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package nodesview

import (
	"fmt"
	"sort"
	"strings"
	"swarmcli/docker"
	"swarmcli/ui"

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

	framed, frame := m.List.RenderFramedView(title)

	if m.labelInputDialog {
		framed = ui.OverlayCentered(framed, m.renderLabelInputDialog(), frame.FrameWidth, frame.FrameHeight)
	} else if m.labelRemoveDialog {
		framed = ui.OverlayCentered(framed, m.renderLabelRemoveDialog(), frame.FrameWidth, frame.FrameHeight)
	} else if m.availabilityDialog {
		framed = ui.OverlayCentered(framed, m.renderAvailabilityDialog(), frame.FrameWidth, frame.FrameHeight)
	} else if m.confirmDialog.Visible {
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
		"ID":           len("ID"),
		"Hostname":     len("HOSTNAME"),
		"Role":         len("ROLE"),
		"State":        len("STATE"),
		"Availability": len("Availability"),
		"Manager":      len("MANAGER"),
		"Version":      len("VERSION"),
		"Addr":         len("ADDRESS"),
		"Labels":       len("LABELS"),
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
		if len(e.Availability) > widths["Availability"] {
			widths["Availability"] = len(e.Availability)
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

// renderAvailabilityDialog renders the availability selection dialog
func (m *Model) renderAvailabilityDialog() string {
	options := []string{"Active", "Pause", "Drain"}
	contentWidth := 40

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("63")).
		Padding(0, 1).
		Width(contentWidth)

	optionStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Width(contentWidth)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("63")).
		Bold(true).
		Padding(0, 2).
		Width(contentWidth)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 2).
		Width(contentWidth)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Width(contentWidth + 2)

	var lines []string
	lines = append(lines, titleStyle.Render(" Set Node Availability "))

	for i, option := range options {
		prefix := "  "
		if i == m.availabilitySelection {
			prefix = "> "
			lines = append(lines, selectedStyle.Render(prefix+option))
		} else {
			lines = append(lines, optionStyle.Render(prefix+option))
		}
	}

	helpText := "↑/↓ Navigate • Enter Confirm • Esc Cancel"
	lines = append(lines, helpStyle.Render(helpText))

	content := strings.Join(lines, "\n")
	return borderStyle.Render(content)
}

// renderLabelInputDialog renders the label input dialog
func (m *Model) renderLabelInputDialog() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("214")).
		Padding(0, 1)

	inputStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Bold(true)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("214")).
		Padding(1, 2)

	var lines []string
	lines = append(lines, titleStyle.Render("Add Node Label"))
	lines = append(lines, "")
	lines = append(lines, inputStyle.Render(m.labelInputValue+"█"))
	lines = append(lines, "")
	lines = append(lines, helpStyle.Render("Format: key=value • Enter Confirm • Esc Cancel"))

	content := strings.Join(lines, "\n")
	return borderStyle.Render(content)
}

// renderLabelRemoveDialog renders the label removal dialog
func (m *Model) renderLabelRemoveDialog() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("214")).
		Padding(0, 1)

	optionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("250"))

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Bold(true)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Italic(true)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("214")).
		Padding(1, 2)

	var lines []string
	lines = append(lines, titleStyle.Render("Remove Node Label"))
	lines = append(lines, "")

	for i, label := range m.labelRemoveLabels {
		prefix := "  "
		if i == m.labelRemoveSelection {
			prefix = "> "
			lines = append(lines, selectedStyle.Render(prefix+label))
		} else {
			lines = append(lines, optionStyle.Render(prefix+label))
		}
	}

	lines = append(lines, "")
	helpText := "↑/↓ Navigate • Enter Confirm • Esc Cancel"
	lines = append(lines, helpStyle.Render(helpText))

	content := strings.Join(lines, "\n")
	return borderStyle.Render(content)
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
