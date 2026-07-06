// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package servicesview

import (
	"swarmcli/ui"
)

func (m *Model) FrameTitle() string  { return m.title }
func (m *Model) FrameHeader() string { return m.List.RenderHeader() }
func (m *Model) FrameFooter() string {
	footer := m.List.RenderFooter()
	if hint := healthFooterHint(); hint != "" {
		// "* " ties the note to the "*" placeholders in the HEALTH column.
		footer += "\n" + ui.StatusBarStyle.Render("* "+hint)
	}
	return footer
}

func (m *Model) FrameContent() string {
	header := m.FrameHeader()
	footer := m.FrameFooter()

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
			m.List.SkipOffsetAdjustment = true

			lineOffset := 0
			for i := 0; i < m.List.Cursor; i++ {
				e := m.List.Filtered[i]
				lineOffset++
				if m.expandedServices[e.ServiceID] {
					tasks := m.serviceTasks[e.ServiceID]
					if len(tasks) > 0 {
						lineOffset += 1 + len(tasks)
					} else {
						lineOffset += 1
					}
				}
			}
			lineOffset += 2 + m.selectedTaskIndex

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
		m.List.SkipOffsetAdjustment = false
	}

	content := m.List.VisibleContent(frame.DesiredContentLines)

	width := frame.FrameWidth
	if m.confirmDialog.Visible {
		content = ui.OverlayCentered(content, m.confirmDialog.View(), width, 0)
	}
	if m.scaleDialog.Visible {
		content = ui.OverlayCentered(content, m.scaleDialog.View(), width, 0)
	}

	return content
}

func (m *Model) View() string {
	return ui.RenderViewFrame(m.FrameTitle(), m.FrameHeader(), m.FrameContent(), m.FrameFooter(),
		m.List.Viewport.Width, m.List.Viewport.Height, false)
}
