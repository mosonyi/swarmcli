package stacksview

import (
	"fmt"
	"strings"
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

	// Compute column widths for 3 columns: STACK | SERVICES | NODES
	width := m.List.Viewport.Width
	if width <= 0 {
		width = m.width
	}
	if width <= 0 {
		width = 80
	}

	cols := 3
	sepLen := 2
	// compute proportional starts to mimic row renderer
	starts := make([]int, cols)
	for i := 0; i < cols; i++ {
		starts[i] = (i * width) / cols
	}
	colWidths := make([]int, cols)
	for i := 0; i < cols; i++ {
		if i == cols-1 {
			colWidths[i] = width - starts[i]
		} else {
			colWidths[i] = starts[i+1] - starts[i]
		}
		if colWidths[i] < 1 {
			colWidths[i] = 1
		}
	}

	// Add separator space back into header widths so labels align with rendered rows
	headerRenderWidths := make([]int, cols)
	for i := 0; i < cols; i++ {
		if i < cols-1 {
			headerRenderWidths[i] = colWidths[i] + sepLen
		} else {
			headerRenderWidths[i] = colWidths[i]
		}
	}
	labels := []string{"  STACK", "SERVICES", "NODES"}
	header := ui.RenderColumnHeader(labels, headerRenderWidths)

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
		if lipgloss.Width(name) > nameMax {
			runes := []rune(name)
			if len(runes) > nameMax {
				if nameMax > 1 {
					name = string(runes[:nameMax-1]) + "…"
				} else {
					name = string(runes[:nameMax])
				}
			}
		}
		first := fmt.Sprintf("  %s", name)

		svcStr := fmt.Sprintf("%d", s.ServiceCount)
		svcMax := colWidths[1]
		if lipgloss.Width(svcStr) > svcMax {
			runes := []rune(svcStr)
			if len(runes) > svcMax {
				svcStr = string(runes[:svcMax])
			}
		}

		nodeStr := fmt.Sprintf("%d", s.NodeCount)
		nodeMax := colWidths[2]
		if lipgloss.Width(nodeStr) > nodeMax {
			runes := []rune(nodeStr)
			if len(runes) > nodeMax {
				nodeStr = string(runes[:nodeMax])
			}
		}

		sep := strings.Repeat(" ", sepLen)
		col0 := fmt.Sprintf("%-*s", colWidths[0], first)
		col1 := fmt.Sprintf("%-*s", colWidths[1], svcStr)
		col2 := fmt.Sprintf("%-*s", colWidths[2], nodeStr)

		line := col0 + sep + col1 + sep + col2

		if selected {
			selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("63")).Bold(true)
			col0 = selStyle.Render(fmt.Sprintf("%-*s", colWidths[0], first) + sep)
			col1 = selStyle.Render(fmt.Sprintf("%-*s", colWidths[1], svcStr) + sep)
			col2 = selStyle.Render(fmt.Sprintf("%-*s", colWidths[2], nodeStr))
			return col0 + col1 + col2
		}
		return lipgloss.NewStyle().Foreground(lipgloss.Color("117")).Render(line)
	}

	// Add 4 to make frame full terminal width (app reduces viewport by 4 in normal mode)
	frameWidth := m.List.Viewport.Width
	if frameWidth <= 0 {
		// Fallback to model width if viewport hasn't been initialized yet
		frameWidth = m.width
	}
	frameWidth = frameWidth + 4

	// Compute frameHeight from viewport (treat Viewport.Height as the total
	// frame height like `configs` view does). Then compute desired inner
	// content lines = frameHeight - borders - header - footer, and pad/trim
	// content to that length.
	// Use the adjusted viewport height directly; the framing helper
	// will account for borders. Do not subtract extra rows here.
	// Reserve two lines from the viewport height for surrounding UI (helpbar/systeminfo)
	frameHeight := m.List.Viewport.Height - 2
	if frameHeight <= 0 {
		// Fallback to model height minus reserved lines if viewport not initialized
		if m.height > 0 {
			frameHeight = m.height - 4
		}
		if frameHeight <= 0 {
			frameHeight = 20
		}
	}

	// Header occupies one line when present (styled header renders single line)
	headerLines := 0
	if header != "" {
		headerLines = 1
	}
	footerLines := 0
	if footer != "" {
		footerLines = len(strings.Split(footer, "\n"))
	}

	desiredContentLines := frameHeight - 2 - headerLines - footerLines
	if desiredContentLines < 0 {
		desiredContentLines = 0
	}

	// Obtain content for exactly `desiredContentLines` rows without mutating
	// the viewport height each render to prevent frame jitter.
	content := m.List.VisibleContent(desiredContentLines)

	contentLines := strings.Split(content, "\n")
	// Trim trailing empty lines
	for len(contentLines) > 0 && contentLines[len(contentLines)-1] == "" {
		contentLines = contentLines[:len(contentLines)-1]
	}
	if len(contentLines) < desiredContentLines {
		for i := 0; i < desiredContentLines-len(contentLines); i++ {
			contentLines = append(contentLines, "")
		}
	} else if len(contentLines) > desiredContentLines {
		contentLines = contentLines[:desiredContentLines]
	}
	paddedContent := strings.Join(contentLines, "\n")

	framed := ui.RenderFramedBoxHeight(title, header, paddedContent, footer, frameWidth, frameHeight)

	return framed
}
