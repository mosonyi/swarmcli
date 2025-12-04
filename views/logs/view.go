package logsview

import (
	"fmt"
	"strings"
	"swarmcli/ui"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}

	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}

	// ---- Build dynamic title bar ----
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

	title := fmt.Sprintf(
		"Service: %s • AutoScroll: %s • wrap: %s • Filter: %s",
		m.ServiceEntry.ServiceName,
		followStatus,
		wrapStatus,
		filterStatus,
	)

	// ---- Build header ----
	header := "Logs"
	if m.mode == "search" {
		header = fmt.Sprintf("Logs — Search: %s", m.searchTerm)
	}

	headerRendered := ui.FrameHeaderStyle.Render(header)

	// ---- Render based on fullscreen mode ----
	if m.fullscreen {
		// In fullscreen: show centered title at top with same styling as frame title
		titleText := ui.FrameTitleStyle.Render(title)
		titleStyle := lipgloss.NewStyle().
			Width(width).
			Align(lipgloss.Center)
		titleRendered := titleStyle.Render(titleText)

		var content string
		// If in search mode, show search header on second line
		if m.mode == "search" {
			searchHeader := ui.FrameHeaderStyle.Render(header)
			content = titleRendered + "\n" + searchHeader + "\n" + m.viewport.View()
		} else {
			content = titleRendered + "\n" + m.viewport.View()
		}

		// Overlay node selection dialog if visible
		if m.getNodeSelectVisible() {
			dialog := m.renderNodeSelectDialog()
			content = ui.OverlayCentered(content, dialog, width, m.viewport.Height+2)
		}

		return content
	}

	// ---- Normal mode: render framed box ----
	content := ui.RenderFramedBox(
		title,
		headerRendered,
		m.viewport.View(),
		"",
		width,
	)

	// Overlay node selection dialog if visible
	if m.getNodeSelectVisible() {
		dialog := m.renderNodeSelectDialog()
		// Calculate actual height by counting lines in rendered content
		contentLines := strings.Split(content, "\n")
		framedHeight := len(contentLines)
		content = ui.OverlayCentered(content, dialog, width, framedHeight)
	}

	return content
}

// renderNodeSelectDialog renders the node selection popup
func (m *Model) renderNodeSelectDialog() string {
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

	for i, node := range nodes {
		if i == cursor {
			lines = append(lines, selectedStyle.Render(" > "+node))
		} else {
			lines = append(lines, itemStyle.Render("   "+node))
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
