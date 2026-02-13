// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package stacksview

import (
	"fmt"
	"strings"
	"swarmcli/ui"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/ui/components/sorting"

	"github.com/charmbracelet/lipgloss"
)

// Shared dialog styles
var (
	dialogTitleStyle = lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("63")).
				Padding(0, 1)

	dialogBorderStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(lipgloss.Color("117"))

	dialogItemStyle = lipgloss.NewStyle().
			Padding(0, 1)

	dialogSelectedStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("15")).
				Background(lipgloss.Color("63")).
				Padding(0, 1)

	dialogHelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1)

	dialogKeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")).
			Bold(true)
)

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}

	title := fmt.Sprintf("Stacks on Node (Total: %d)", len(m.List.Items))

	// Compute four percentage-based column widths so columns start at
	// 0%, 25%, 50%, 75% of the available content width.
	width := m.List.Viewport.Width
	if width <= 0 {
		width = m.width
	}
	if width <= 0 {
		width = 80
	}
	contentWidth := width

	// Calculate column widths: allocate space for stack, services, tasks and error
	// STACK: 25%, SERVICES: 10%, TASKS: 10%, ERROR: 55% (remainder)
	colWidths := make([]int, 4)
	colWidths[0] = (contentWidth * 25) / 100
	colWidths[1] = (contentWidth * 10) / 100
	colWidths[2] = (contentWidth * 10) / 100
	colWidths[3] = contentWidth - colWidths[0] - colWidths[1] - colWidths[2]

	// Build header using frame header style so it appears on the first
	// line inside the framed box and aligns with rows below.
	stackLabel := " STACK"
	if m.sortField == SortByName {
		arrow := sorting.SortArrow(sorting.Ascending)
		if !m.sortAscending {
			arrow = sorting.SortArrow(sorting.Descending)
		}
		stackLabel = fmt.Sprintf(" STACK %s", arrow)
	}

	servicesLabel := "SERVICES"
	if m.sortField == SortByServices {
		arrow := sorting.SortArrow(sorting.Ascending)
		if !m.sortAscending {
			arrow = sorting.SortArrow(sorting.Descending)
		}
		servicesLabel = fmt.Sprintf("SERVICES %s", arrow)
	}

	tasksLabel := "TASKS"
	if m.sortField == SortByTasks {
		arrow := sorting.SortArrow(sorting.Ascending)
		if !m.sortAscending {
			arrow = sorting.SortArrow(sorting.Descending)
		}
		tasksLabel = fmt.Sprintf("TASKS %s", arrow)
	}

	// Add ERROR column after TASKS with count of stacks having errors
	errorCount := 0
	for _, hasErr := range m.stackHasError {
		if hasErr {
			errorCount++
		}
	}
	var errorLabel string
	if m.sortField == SortByError {
		arrow := sorting.SortArrow(sorting.Ascending)
		if !m.sortAscending {
			arrow = sorting.SortArrow(sorting.Descending)
		}
		errorLabel = fmt.Sprintf("ERROR: %d %s", errorCount, arrow)
	} else {
		errorLabel = fmt.Sprintf("ERROR: %d", errorCount)
	}
	headerLine := fmt.Sprintf("%-*s%-*s%-*s%-*s",
		colWidths[0], stackLabel,
		colWidths[1], servicesLabel,
		colWidths[2], tasksLabel,
		colWidths[3], errorLabel,
	)
	header := ui.FrameHeaderStyle.Render(headerLine)

	// Footer: cursor + optional search query
	status := fmt.Sprintf("Stack %d of %d", m.List.Cursor+1, len(m.List.Filtered))
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

	// Ensure RenderItem can include expanded inline tasks
	m.setRenderItem()

	// Compute consistent frame sizing using shared helper (stacks is template)
	frame := ui.ComputeFrameDimensions(
		m.List.Viewport.Width,
		m.List.Viewport.Height,
		m.width,
		m.height,
		header,
		footer,
	)

	// Use VisibleContent to get only the visible portion based on cursor position
	// This ensures proper scrolling and that the cursor is always visible
	// VisibleContent already returns exactly desiredContentLines, so we use
	// RenderFramedBox instead of RenderFramedBoxHeight to avoid double-padding
	content := m.List.VisibleContent(frame.DesiredContentLines)

	framed := ui.RenderFramedBox(title, header, content, footer, frame.FrameWidth)

	// Debug: log dialog states
	l().Debugf("Rendering stacks view: createDialogActive=%v step=%s fileBrowserActive=%v",
		m.createDialogActive, m.createDialogStep, m.fileBrowserActive)

	if m.createDialogActive {
		framed = ui.OverlayCentered(framed, m.renderCreateDialog(), frame.FrameWidth, frame.FrameHeight)
	} else if m.fileBrowserActive {
		fileBrowserDialog := ui.RenderFileBrowserDialog("Select Compose File", m.fileBrowserPath, m.fileBrowserFiles, m.fileBrowserCursor)
		framed = ui.OverlayCentered(framed, fileBrowserDialog, frame.FrameWidth, frame.FrameHeight)
	} else if m.confirmDialog.Visible {
		framed = ui.OverlayCentered(framed, m.confirmDialog.View(), frame.FrameWidth, frame.FrameHeight)
	}

	return framed
}

func (m *Model) renderCreateDialog() string {
	var lines []string

	l().Debugf("renderCreateDialog: step=%s content=%d bytes path=%s",
		m.createDialogStep, len(m.createDialogContent), m.createStackPath)

	switch m.createDialogStep {
	case "source":
		lines = append(lines, dialogTitleStyle.Render(" Create Stack - Choose Source "))
		lines = append(lines, dialogItemStyle.Render(""))
		lines = append(lines, dialogItemStyle.Render("How would you like to create the stack?"))
		lines = append(lines, dialogItemStyle.Render(""))

		if m.createStackSource == "file" {
			lines = append(lines, dialogSelectedStyle.Render("→ From compose file"))
		} else {
			lines = append(lines, dialogItemStyle.Render("  From compose file"))
		}

		if m.createStackSource == "inline" {
			lines = append(lines, dialogSelectedStyle.Render("→ Inline editor"))
		} else {
			lines = append(lines, dialogItemStyle.Render("  Inline editor"))
		}

		lines = append(lines, dialogItemStyle.Render(""))
		helpText := fmt.Sprintf(" %s Select • %s / %s Navigate • %s Cancel",
			dialogKeyStyle.Render("<Enter>"),
			dialogKeyStyle.Render("<↑>"),
			dialogKeyStyle.Render("<↓>"),
			dialogKeyStyle.Render("<Esc>"))
		lines = append(lines, dialogHelpStyle.Render(helpText))

	case "details-file":
		lines = append(lines, dialogTitleStyle.Render(" Create Stack from Compose File "))
		lines = append(lines, dialogItemStyle.Render(""))

		// Show error if present
		if m.createDialogError != "" {
			errorStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Padding(0, 1).
				Width(70)
			// Wrap error message to multiple lines if needed
			wrappedError := wrapText("⚠ "+m.createDialogError, 68) // 70 - 2 for padding
			for _, line := range wrappedError {
				lines = append(lines, errorStyle.Render(line))
			}
			lines = append(lines, dialogItemStyle.Render(""))
		}

		lines = append(lines, dialogItemStyle.Render(m.createNameInput.View()))
		lines = append(lines, dialogItemStyle.Render(""))

		// Show file path input with browse indicator when focused
		// Always reserve space for the hint to prevent dialog resizing
		fileLine := m.createFileInput.View()
		if m.createInputFocus == 1 {
			fileLine += "  " + dialogKeyStyle.Render("[f: Browse]")
		} else {
			// Add invisible spacing to maintain consistent width
			fileLine += "             " // 13 characters to match "  [f: Browse]"
		}
		lines = append(lines, dialogItemStyle.Render(fileLine))
		lines = append(lines, dialogItemStyle.Render(""))

		// Change help text based on error state
		var helpText string
		if m.createDialogError != "" {
			helpText = fmt.Sprintf(" %s Fix error • %s Navigate • %s Cancel",
				dialogKeyStyle.Render("<Enter>"),
				dialogKeyStyle.Render("<Tab>"),
				dialogKeyStyle.Render("<Esc>"))
		} else {
			helpText = fmt.Sprintf(" %s Deploy • %s Navigate • %s Browse • %s Cancel",
				dialogKeyStyle.Render("<Enter>"),
				dialogKeyStyle.Render("<Tab>"),
				dialogKeyStyle.Render("<f>"),
				dialogKeyStyle.Render("<Esc>"))
		}
		lines = append(lines, dialogHelpStyle.Render(helpText))

	case "details-inline":
		lines = append(lines, dialogTitleStyle.Render(" Create Stack - Inline Editor "))
		lines = append(lines, dialogItemStyle.Render(""))

		// Show error if present
		if m.createDialogError != "" {
			errorStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("196")).
				Padding(0, 1).
				Width(70)
			// Wrap error message to multiple lines if needed
			wrappedError := wrapText("⚠ "+m.createDialogError, 68) // 70 - 2 for padding
			for _, line := range wrappedError {
				lines = append(lines, errorStyle.Render(line))
			}
			lines = append(lines, dialogItemStyle.Render(""))
		}

		lines = append(lines, dialogItemStyle.Render(m.createNameInput.View()))
		lines = append(lines, dialogItemStyle.Render(""))

		// Show source file if content was loaded from a file
		if m.createStackPath != "" {
			sourceStyle := lipgloss.NewStyle().
				Foreground(lipgloss.Color("240")).
				Padding(0, 1)
			lines = append(lines, sourceStyle.Render(fmt.Sprintf("Source: %s", m.createStackPath)))
			lines = append(lines, dialogItemStyle.Render(""))
		}

		// Show editor status with edit hint when focused
		// Always reserve space for the hint to prevent dialog resizing
		editorStatus := "Compose YAML: "
		if m.createDialogContent != "" {
			editorStatus += fmt.Sprintf("(%d bytes)", len(m.createDialogContent))
		} else {
			editorStatus += "(empty)"
		}
		if m.createInputFocus == 1 {
			editorStatus += "  " + dialogKeyStyle.Render("[e: Edit]")
		} else {
			// Add invisible spacing to maintain consistent width
			editorStatus += "           " // 11 characters to match "  [e: Edit]"
		}
		lines = append(lines, dialogItemStyle.Render(editorStatus))
		lines = append(lines, dialogItemStyle.Render(""))

		// Change help text based on error state
		var helpText string
		if m.createDialogError != "" {
			helpText = fmt.Sprintf(" %s Fix error • %s Navigate • %s Cancel",
				dialogKeyStyle.Render("<Enter>"),
				dialogKeyStyle.Render("<Tab>"),
				dialogKeyStyle.Render("<Esc>"))
		} else {
			helpText = fmt.Sprintf(" %s Deploy • %s Navigate • %s Edit • %s Cancel",
				dialogKeyStyle.Render("<Enter>"),
				dialogKeyStyle.Render("<Tab>"),
				dialogKeyStyle.Render("<e>"),
				dialogKeyStyle.Render("<Esc>"))
		}
		lines = append(lines, dialogHelpStyle.Render(helpText))
	}

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return dialogBorderStyle.Render(content)
}

// wrapText wraps text to the specified width, breaking on word boundaries
func wrapText(text string, width int) []string {
	if len(text) <= width {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{text}
	}

	currentLine := ""
	for _, word := range words {
		// If adding this word would exceed width
		if len(currentLine)+len(word)+1 > width {
			if currentLine != "" {
				lines = append(lines, currentLine)
				currentLine = word
			} else {
				// Word itself is longer than width, just add it
				lines = append(lines, word)
			}
		} else {
			if currentLine == "" {
				currentLine = word
			} else {
				currentLine += " " + word
			}
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}
