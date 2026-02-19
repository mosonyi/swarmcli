// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package servicesview

import (
	"swarmcli/ui"
)

func (m *Model) View() string {
	header := m.List.RenderHeader()
	footer := m.List.RenderFooter()

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
