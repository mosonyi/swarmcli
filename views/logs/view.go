// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package logsview

import (
	"fmt"
	"swarmcli/ui"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) FrameTitle() string {
	followStatus := "off"
	if m.getFollow() {
		followStatus = "on"
	}
	wrapStatus := "off"
	if m.getWrap() {
		wrapStatus = "on"
	}
	nodeFilter := m.getNodeFilter()
	filterStatus := "all nodes"
	if nodeFilter != "" {
		filterStatus = fmt.Sprintf("node: %s", nodeFilter)
	}
	stoppedStatus := "hidden"
	if !m.getHideStopped() {
		stoppedStatus = "shown"
	}
	return fmt.Sprintf(
		"Service: %s • AutoScroll: %s • wrap: %s • Filter: %s • Stopped: %s",
		m.ServiceEntry.ServiceName,
		followStatus,
		wrapStatus,
		filterStatus,
		stoppedStatus,
	)
}

func (m *Model) FrameHeader() string {
	header := "Logs"
	if m.mode == "search" {
		header = fmt.Sprintf("Logs — Search: %s", m.searchTerm)
	} else if m.searchTerm != "" && len(m.searchMatches) > 0 {
		matchCount := len(m.searchMatches)
		currentMatch := m.searchIndex + 1
		header = fmt.Sprintf("Logs — Found %d matches (viewing %d/%d) • Press 'ctrl+f' to search, 'n'/'N' to navigate", matchCount, currentMatch, matchCount)
	}
	return ui.FrameHeaderStyle.Render(header)
}

func (m *Model) FrameFooter() string { return "" }

func (m *Model) FrameContent() string {
	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}

	headerRendered := m.FrameHeader()
	frame := ui.ComputeFrameDimensions(
		width, m.viewport.Height,
		width, m.viewport.Height,
		headerRendered, "",
	)

	viewportContent := ui.TrimOrPadContentToLines(m.viewport.View(), frame.DesiredContentLines)

	if m.getNodeSelectVisible() && m.viewport.Height >= 5 {
		availableHeight := m.viewport.Height
		dialog := m.renderNodeSelectDialog(availableHeight)
		viewportContent = ui.OverlayCentered(viewportContent, dialog, width, 0)
	}

	return viewportContent
}

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}
	return ui.RenderViewFrame(m.FrameTitle(), m.FrameHeader(), m.FrameContent(), m.FrameFooter(),
		m.viewport.Width, m.viewport.Height, false)
}

// renderNodeSelectDialog renders the node selection popup
func (m *Model) renderNodeSelectDialog(availableHeight int) string {
	// Lock to safely access dialog state
	m.mu.Lock()
	nodes := make([]string, len(m.nodeSelectNodes))
	copy(nodes, m.nodeSelectNodes)
	cursor := m.nodeSelectCursor
	m.mu.Unlock()

	// Safety check: if no nodes, return empty string
	if len(nodes) == 0 {
		return ""
	}

	// Ensure cursor is within bounds
	if cursor < 0 {
		cursor = 0
	}
	if cursor >= len(nodes) {
		cursor = len(nodes) - 1
	}

	// Calculate required width based on help text
	helpTextPlain := fmt.Sprintf(" %s Navigate • %s Select • %s Cancel",
		"<↑/↓/PgUp/PgDn>",
		"<Enter>",
		"<Esc>")
	helpTextMinWidth := lipgloss.Width(helpTextPlain) + 2 // add padding

	// Set minimum width to accommodate help text
	contentWidth := helpTextMinWidth
	if contentWidth < 40 {
		contentWidth = 40 // minimum for usability
	}

	// Check if any node names are longer
	for _, node := range nodes {
		nodeWidth := lipgloss.Width(node) + 4 // " > " prefix + space
		if nodeWidth > contentWidth {
			contentWidth = nodeWidth
		}
	}

	titleWidth := lipgloss.Width(" Select Node to Filter ")
	if titleWidth > contentWidth {
		contentWidth = titleWidth
	}

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("63")).
		Padding(0, 1).
		Width(contentWidth)

	itemStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Width(contentWidth)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("63")).
		Bold(true).
		Padding(0, 1).
		Width(contentWidth)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("117")).
		Width(contentWidth + 2)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(contentWidth)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("63")).
		Bold(true)

	// Build the content
	var lines []string
	lines = append(lines, titleStyle.Render(" Select Node to Filter "))

	// Calculate visible window for scrollable list
	maxVisibleItems := availableHeight - 4
	if maxVisibleItems < 1 {
		maxVisibleItems = 1
	}

	totalItems := len(nodes)

	// If all items fit, show them all
	if totalItems <= maxVisibleItems {
		for i, node := range nodes {
			if i == cursor {
				lines = append(lines, selectedStyle.Render(" > "+node))
			} else {
				lines = append(lines, itemStyle.Render("   "+node))
			}
		}
	} else {
		// Scrolling needed - calculate visible window
		effectiveVisibleItems := maxVisibleItems

		halfWindow := effectiveVisibleItems / 2
		startIdx := cursor - halfWindow
		if startIdx < 0 {
			startIdx = 0
		}
		endIdx := startIdx + effectiveVisibleItems
		if endIdx > totalItems {
			endIdx = totalItems
			startIdx = endIdx - effectiveVisibleItems
			if startIdx < 0 {
				startIdx = 0
			}
		}

		linesAdded := 0

		if startIdx > 0 {
			indicatorLine := fmt.Sprintf("   ↑ %d more above", startIdx)
			lines = append(lines, itemStyle.Render(indicatorLine))
			linesAdded++
		}

		remainingSlots := maxVisibleItems - linesAdded
		if endIdx > totalItems {
			endIdx = totalItems
		}
		if endIdx < totalItems {
			remainingSlots--
		}

		actualEndIdx := startIdx + remainingSlots
		if actualEndIdx > totalItems {
			actualEndIdx = totalItems
		}

		for i := startIdx; i < actualEndIdx; i++ {
			node := nodes[i]
			if i == cursor {
				lines = append(lines, selectedStyle.Render(" > "+node))
			} else {
				lines = append(lines, itemStyle.Render("   "+node))
			}
			linesAdded++
		}

		if actualEndIdx < totalItems {
			indicatorLine := fmt.Sprintf("   ↓ %d more below", totalItems-actualEndIdx)
			lines = append(lines, itemStyle.Render(indicatorLine))
		}
	}

	// Build help text with styled keys
	helpText := fmt.Sprintf(" %s Navigate • %s Select • %s Cancel",
		keyStyle.Render("<↑/↓/PgUp/PgDn>"),
		keyStyle.Render("<Enter>"),
		keyStyle.Render("<Esc>"))
	lines = append(lines, helpStyle.Render(helpText))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return borderStyle.Render(content)
}
