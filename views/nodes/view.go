// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package nodesview

import (
	"fmt"
	"sort"
	"strings"
	"swarmcli/ui"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) FrameTitle() string {
	total := len(m.List.Items)
	managers := 0
	for _, n := range m.List.Items {
		if n.Manager {
			managers++
		}
	}
	return fmt.Sprintf("Nodes (%d total, %d manager%s)", total, managers, plural(managers))
}

func (m *Model) FrameHeader() string { return m.List.RenderHeader() }
func (m *Model) FrameFooter() string { return m.List.RenderFooter() }

func (m *Model) FrameContent() string {
	header := m.FrameHeader()
	footer := m.FrameFooter()
	frame := ui.ComputeFrameDimensions(
		m.List.Viewport.Width, m.List.Viewport.Height,
		m.List.Viewport.Width, m.List.Viewport.Height,
		header, footer,
	)
	content := m.List.VisibleContent(frame.DesiredContentLines)

	width := frame.FrameWidth
	if m.labelInputDialog {
		content = ui.OverlayCentered(content, m.renderLabelInputDialog(), width, 0)
	} else if m.labelRemoveDialog {
		content = ui.OverlayCentered(content, m.renderLabelRemoveDialog(), width, 0)
	} else if m.availabilityDialog {
		content = ui.OverlayCentered(content, m.renderAvailabilityDialog(), width, 0)
	} else if m.confirmDialog.Visible {
		content = ui.OverlayCentered(content, m.confirmDialog.View(), width, 0)
	}

	return content
}

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}
	return ui.RenderViewFrame(m.FrameTitle(), m.FrameHeader(), m.FrameContent(), m.FrameFooter(),
		m.List.Viewport.Width, m.List.Viewport.Height, false)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
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
