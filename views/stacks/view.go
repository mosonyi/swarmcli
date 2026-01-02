package stacksview

import (
	"fmt"
	"swarmcli/docker"
	"swarmcli/ui"
	filterlist "swarmcli/ui/components/filterable/list"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}

	title := fmt.Sprintf("Stacks on Node (Total: %d)", len(m.List.Items))

	// Compute three percentage-based column widths so columns start at
	// 0%, 33%, 66% of the available content width.
	width := m.List.Viewport.Width
	if width <= 0 {
		width = m.width
	}
	if width <= 0 {
		width = 80
	}
	contentWidth := width

	// Calculate column widths: each column gets 33% of width
	colWidths := make([]int, 3)
	colWidths[0] = (contentWidth * 33) / 100
	colWidths[1] = (contentWidth * 33) / 100
	colWidths[2] = contentWidth - colWidths[0] - colWidths[1] // Remaining width for last column

	// Build header using frame header style so it appears on the first
	// line inside the framed box and aligns with rows below.
	headerLine := fmt.Sprintf("%-*s%-*s%-*s",
		colWidths[0], "  STACK",
		colWidths[1], "SERVICES",
		colWidths[2], "NODES",
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

	// Set RenderItem to format rows using the same colWidths so the
	// header and rows align exactly.
	m.List.RenderItem = func(s docker.StackEntry, selected bool, _ int) string {
		// First column: current marker + name (we don't have a marker here but keep spacing)
		nameMax := colWidths[0] - 2
		if nameMax < 0 {
			nameMax = 0
		}
		name := s.Name
		if len(name) > nameMax {
			if nameMax > 3 {
				name = name[:nameMax-3] + "..."
			} else {
				name = name[:nameMax]
			}
		}
		first := fmt.Sprintf("  %s", name)

		svcStr := fmt.Sprintf("%d", s.ServiceCount)
		svcMax := colWidths[1]
		if len(svcStr) > svcMax {
			svcStr = svcStr[:svcMax]
		}

		nodeStr := fmt.Sprintf("%d", s.NodeCount)
		nodeMax := colWidths[2]
		if len(nodeStr) > nodeMax {
			nodeStr = nodeStr[:nodeMax]
		}

		line := fmt.Sprintf("%-*s%-*s%-*s",
			colWidths[0], first,
			colWidths[1], svcStr,
			colWidths[2], nodeStr,
		)
		if selected {
			selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("63")).Bold(true)
			return selStyle.Render(line)
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("15")).Render(line)
	}

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

	return framed
}
