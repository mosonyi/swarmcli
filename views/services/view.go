// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package servicesview

import (
	"fmt"
	"swarmcli/ui"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/ui/components/sorting"
)

func (m *Model) View() string {
	width := m.List.Viewport.Width
	if width <= 0 {
		width = 80
	}

	// The header column widths are computed further down using the same
	// effective-width logic as the renderer; see that computation below.
	labels := []string{" SERVICE", "STACK", "REPLICAS", "STATUS", "MODE", "IMAGE", "PORTS", "CREATED", "UPDATED", "ERROR"}

	// Add sort indicators to labels
	if m.sortField == SortByName {
		arrow := sorting.SortArrow(sorting.Ascending)
		if !m.sortAscending {
			arrow = sorting.SortArrow(sorting.Descending)
		}
		labels[0] = fmt.Sprintf(" SERVICE %s", arrow)
	}
	if m.sortField == SortByStatus {
		arrow := sorting.SortArrow(sorting.Ascending)
		if !m.sortAscending {
			arrow = sorting.SortArrow(sorting.Descending)
		}
		labels[3] = fmt.Sprintf("STATUS %s", arrow)
	}
	if m.sortField == SortByImage {
		arrow := sorting.SortArrow(sorting.Ascending)
		if !m.sortAscending {
			arrow = sorting.SortArrow(sorting.Descending)
		}
		labels[5] = fmt.Sprintf("IMAGE %s", arrow)
	}
	if m.sortField == SortByPorts {
		arrow := sorting.SortArrow(sorting.Ascending)
		if !m.sortAscending {
			arrow = sorting.SortArrow(sorting.Descending)
		}
		labels[6] = fmt.Sprintf("PORTS %s", arrow)
	}
	if m.sortField == SortByCreated {
		arrow := sorting.SortArrow(sorting.Ascending)
		if !m.sortAscending {
			arrow = sorting.SortArrow(sorting.Descending)
		}
		labels[7] = fmt.Sprintf("CREATED %s", arrow)
	}
	if m.sortField == SortByUpdated {
		arrow := sorting.SortArrow(sorting.Ascending)
		if !m.sortAscending {
			arrow = sorting.SortArrow(sorting.Descending)
		}
		labels[8] = fmt.Sprintf("UPDATED %s", arrow)
	}
	if m.sortField == SortByError {
		arrow := sorting.SortArrow(sorting.Ascending)
		if !m.sortAscending {
			arrow = sorting.SortArrow(sorting.Descending)
		}
		labels[9] = fmt.Sprintf("ERROR %s", arrow)
	}
	// Use the same column width computation as the renderer so header
	// and rows stay perfectly aligned.
	colWidths := m.computeColWidths(width)
	headerLine := m.buildHeaderLine(labels, colWidths)
	header := ui.FrameHeaderStyle.Render(headerLine)

	// Footer: cursor + optional search query
	status := fmt.Sprintf("Node %d of %d", m.List.Cursor+1, len(m.List.Filtered))
	statusBar := ui.StatusBarStyle.Render(status)

	var footer string
	if m.List.Mode == filterlist.ModeSearching {
		footer = ui.StatusBarStyle.Render("Filter (type then Enter): " + m.List.Query)
	} else if m.List.Query != "" {
		footer = ui.StatusBarStyle.Render("Filter: " + m.List.Query)
	}

	// Compose footer (status bar + optional filter line)
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

	// Adjust viewport offset for task navigation before calling VisibleContent
	if m.selectedTaskIndex >= 0 && m.List.Cursor < len(m.List.Filtered) {
		entry := m.List.Filtered[m.List.Cursor]
		if m.expandedServices[entry.ServiceID] {
			// Skip VisibleContent's offset adjustment - we'll manage it manually
			m.List.SkipOffsetAdjustment = true

			// Calculate the line offset for the selected task
			lineOffset := 0
			for i := 0; i < m.List.Cursor; i++ {
				e := m.List.Filtered[i]
				lineOffset++ // service row
				if m.expandedServices[e.ServiceID] {
					tasks := m.serviceTasks[e.ServiceID]
					if len(tasks) > 0 {
						lineOffset += 1 + len(tasks) // header + task rows
					} else {
						lineOffset += 1 // "no tasks" row
					}
				}
			}
			// Add service row + header + selected task
			lineOffset += 2 + m.selectedTaskIndex

			// Ensure the task line is visible
			if lineOffset < m.List.Viewport.YOffset {
				m.List.Viewport.YOffset = lineOffset
			} else if lineOffset >= m.List.Viewport.YOffset+frame.DesiredContentLines {
				m.List.Viewport.YOffset = lineOffset - frame.DesiredContentLines + 1
				if m.List.Viewport.YOffset < 0 {
					m.List.Viewport.YOffset = 0
				}
			}
		}
	} else {
		// Not in task navigation - let VisibleContent handle offset
		m.List.SkipOffsetAdjustment = false
	}

	// Use VisibleContent to get only the visible portion based on cursor position
	// This ensures proper scrolling and that the cursor is always visible
	// VisibleContent already returns exactly desiredContentLines, so we use
	// RenderFramedBox instead of RenderFramedBoxHeight to avoid double-padding
	content := m.List.VisibleContent(frame.DesiredContentLines)

	framed := ui.RenderFramedBox(m.title, header, content, footer, frame.FrameWidth)

	if m.confirmDialog.Visible {
		framed = ui.OverlayCentered(framed, m.confirmDialog.View(), frame.FrameWidth, frame.FrameHeight)
	}

	if m.scaleDialog.Visible {
		framed = ui.OverlayCentered(framed, m.scaleDialog.View(), frame.FrameWidth, frame.FrameHeight)
	}

	return framed
}
