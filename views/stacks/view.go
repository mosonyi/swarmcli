// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package stacksview

import (
	"fmt"
	"swarmcli/ui"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/ui/components/sorting"
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

	if m.confirmDialog.Visible {
		framed = ui.OverlayCentered(framed, m.confirmDialog.View(), frame.FrameWidth, frame.FrameHeight)
	}

	return framed
}
