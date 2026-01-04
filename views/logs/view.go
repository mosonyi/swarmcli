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
			availableHeight := m.viewport.Height
			if availableHeight < 7 {
				availableHeight = 7
			}
			dialog := m.renderNodeSelectDialog(availableHeight)
			content = ui.OverlayCentered(content, dialog, width, m.viewport.Height+2)
		}

		return content
	}

	// ---- Normal mode: render framed box ----
	frame := ui.ComputeFrameDimensions(
		width,             // viewport width (already minus 4 from app)
		m.viewport.Height, // adjusted height from app
		width,             // fallback width
		m.viewport.Height, // fallback height (viewport stores last known height)
		headerRendered,
		"",
	)

	// Get viewport content and truncate to fit the frame
	viewportContent := ui.TrimOrPadContentToLines(m.viewport.View(), frame.DesiredContentLines)

	// If dialog is visible, overlay it on the viewport content BEFORE framing
	if m.getNodeSelectVisible() {
		// Use actual viewport height for dialog, ensuring it fits
		availableHeight := m.viewport.Height
		// Only render dialog if we have minimum space (at least 5 lines)
		if m.viewport.Height >= 5 {
			dialog := m.renderNodeSelectDialog(availableHeight)

			// Manual overlay: preserve background and place dialog on top
			viewportLines := strings.Split(viewportContent, "\n")
			dialogLines := strings.Split(dialog, "\n")

			// Calculate dialog position (centered)
			dialogHeight := len(dialogLines)
			dialogWidth := 0
			for _, line := range dialogLines {
				if w := lipgloss.Width(line); w > dialogWidth {
					dialogWidth = w
				}
			}

			startRow := (len(viewportLines) - dialogHeight) / 2
			if startRow < 0 {
				startRow = 0
			}

			startCol := (width - dialogWidth) / 2
			if startCol < 0 {
				startCol = 0
			}

			// IMPORTANT: RenderFramedBox will add padding (borderWidth = frameWidth - 2 = width + 2)
			// This means content gets padded from 'width' to 'width + 2'
			// The padding is added at the END, so our positions are correct
			// BUT we calculated startCol based on 'width', which is correct for viewport
			// No adjustment needed since padding is at the end, not distributed

			// Overlay dialog lines onto viewport lines
			for i, dialogLine := range dialogLines {
				row := startRow + i
				if row < 0 || row >= len(viewportLines) {
					continue
				}

				baseLine := viewportLines[row]

				// Work with visual widths and string slicing, accounting for ANSI codes
				// We need to find the byte positions that correspond to visual positions
				baseVisualWidth := lipgloss.Width(baseLine)
				dialogVisualWidth := lipgloss.Width(dialogLine)

				// Build new line preserving content around dialog
				var newLine strings.Builder

				// If base line is shorter than where dialog should start, pad it
				if baseVisualWidth < startCol {
					newLine.WriteString(baseLine)
					newLine.WriteString(strings.Repeat(" ", startCol-baseVisualWidth))
					newLine.WriteString(dialogLine)
				} else {
					// Need to extract left part (visual width = startCol)
					// and right part (starting at visual position startCol + dialogVisualWidth)
					leftPart := ""
					rightPart := ""

					// Extract left part up to startCol visual width
					currentVisualPos := 0
					currentBytePos := 0
					inAnsi := false

					for currentBytePos < len(baseLine) && currentVisualPos < startCol {
						if baseLine[currentBytePos] == '\x1b' && currentBytePos+1 < len(baseLine) && baseLine[currentBytePos+1] == '[' {
							inAnsi = true
						}
						if !inAnsi {
							currentVisualPos++
						}
						currentBytePos++
						if inAnsi && currentBytePos < len(baseLine) && baseLine[currentBytePos-1] == 'm' {
							inAnsi = false
						}
					}
					leftPart = baseLine[:currentBytePos]

					// Extract right part starting at startCol + dialogVisualWidth
					targetVisualPos := startCol + dialogVisualWidth
					currentVisualPos = 0
					currentBytePos = 0
					inAnsi = false

					for currentBytePos < len(baseLine) && currentVisualPos < targetVisualPos {
						if baseLine[currentBytePos] == '\x1b' && currentBytePos+1 < len(baseLine) && baseLine[currentBytePos+1] == '[' {
							inAnsi = true
						}
						if !inAnsi {
							currentVisualPos++
						}
						currentBytePos++
						if inAnsi && currentBytePos < len(baseLine) && baseLine[currentBytePos-1] == 'm' {
							inAnsi = false
						}
					}
					if currentBytePos < len(baseLine) {
						rightPart = baseLine[currentBytePos:]
					}

					newLine.WriteString(leftPart)
					newLine.WriteString(dialogLine)
					newLine.WriteString(rightPart)
				}

				viewportLines[row] = newLine.String()
			}

			viewportContent = strings.Join(viewportLines, "\n")
		}
	}

	content := ui.RenderFramedBox(
		title,
		headerRendered,
		viewportContent,
		"",
		frame.FrameWidth,
	)

	return content
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
	// Available height includes: border top (1) + title (1) + items + help (1) + border bottom (1) = 4 fixed
	// So available for items = availableHeight - 4
	// But if we need to show "more above/below" indicators, those take item slots too
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
		// Reserve space for indicators if needed
		effectiveVisibleItems := maxVisibleItems

		// Keep cursor in view with some context
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

		// Track how many lines we've added
		linesAdded := 0

		// Show indicator if there are items above
		if startIdx > 0 {
			indicatorLine := fmt.Sprintf("   ↑ %d more above", startIdx)
			lines = append(lines, itemStyle.Render(indicatorLine))
			linesAdded++
		}

		// Show visible items (adjust count to fit within maxVisibleItems including indicators)
		remainingSlots := maxVisibleItems - linesAdded
		if endIdx > totalItems {
			endIdx = totalItems
		}
		// Reserve 1 slot for "more below" indicator if needed
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

		// Show indicator if there are items below
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
