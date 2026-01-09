package servicesview

import (
	"fmt"
	"swarmcli/docker"
	"swarmcli/ui"
	filterlist "swarmcli/ui/components/filterable/list"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	width := m.List.Viewport.Width
	if width <= 0 {
		width = 80
	}

	// The header column widths are computed further down using the same
	// effective-width logic as the renderer; see that computation below.
	cols := 6

	labels := []string{" SERVICE", "STACK", "REPLICAS", "STATUS", "CREATED", "UPDATED"}
	// Compute header column widths using the same effective-width logic
	// (columns widths exclude two-space separators) so the header aligns
	// exactly with the data columns.
	sepLen := 2
	sepTotal := sepLen * (cols - 1)
	effWidth := width - sepTotal
	if effWidth < cols {
		effWidth = width
	}

	// Reuse same minCols and longest service logic as RenderItem would
	minCols := make([]int, cols)
	for i := 0; i < cols; i++ {
		hw := lipgloss.Width(labels[i])
		floor := 6
		switch i {
		case 0:
			floor = 10
		case 1:
			floor = 10
		case 2:
			floor = 8
		case 3:
			floor = 8
		case 4, 5:
			floor = 8
		}
		if hw > floor {
			minCols[i] = hw
		} else {
			minCols[i] = floor
		}
	}

	maxSvc := lipgloss.Width(labels[0])
	for _, it := range m.List.Items {
		if s, ok := any(it).(docker.ServiceEntry); ok {
			if w := lipgloss.Width(s.ServiceName); w > maxSvc {
				maxSvc = w
			}
		}
	}
	desiredSvc := maxSvc + 1

	headerColWidths := make([]int, cols)
	nonServiceMinSum := 0
	for i := 1; i < cols; i++ {
		nonServiceMinSum += minCols[i]
	}
	if desiredSvc+nonServiceMinSum <= effWidth {
		headerColWidths[0] = desiredSvc
		for i := 1; i < cols; i++ {
			headerColWidths[i] = minCols[i]
		}
		// distribute leftover across cols 1..5
		sum := 0
		for _, v := range headerColWidths {
			sum += v
		}
		leftover := effWidth - sum
		if leftover > 0 {
			per := leftover / (cols - 1)
			rem := leftover % (cols - 1)
			for i := 1; i < cols; i++ {
				add := per
				if rem > 0 {
					add++
					rem--
				}
				headerColWidths[i] += add
			}
		}
	} else {
		base := effWidth / cols
		for i := 0; i < cols; i++ {
			headerColWidths[i] = base
			if headerColWidths[i] < minCols[i] {
				headerColWidths[i] = minCols[i]
			}
		}
		sum := 0
		for _, v := range headerColWidths {
			sum += v
		}
		if sum != effWidth {
			headerColWidths[cols-1] += effWidth - sum
		}
	}

	// Convert effective column widths into header render widths. The item
	// renderer formats most columns with a `-1` width (to reserve a char),
	// then appends separators of length `sepLen`. Recreate the exact visual
	// width here so header labels align with data.
	headerRenderWidths := make([]int, cols)
	for i := 0; i < cols; i++ {
		if i == 0 {
			// first column uses full effective width then separator
			headerRenderWidths[i] = headerColWidths[i] + sepLen
		} else if i < cols-1 {
			// non-first, non-last columns are rendered with colWidths[i]-1
			// plus separator
			w := headerColWidths[i]
			if w > 0 {
				w = w - 1
			}
			headerRenderWidths[i] = w + sepLen
			if headerRenderWidths[i] < 1 {
				headerRenderWidths[i] = 1
			}
		} else {
			// last column uses its full width (no trailing separator)
			headerRenderWidths[i] = headerColWidths[i]
			if headerRenderWidths[i] < 1 {
				headerRenderWidths[i] = 1
			}
		}
	}
	header := ui.RenderColumnHeader(labels, headerRenderWidths)

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

	// Use VisibleContent to get only the visible portion based on cursor position
	// This ensures proper scrolling and that the cursor is always visible
	// VisibleContent already returns exactly desiredContentLines, so we use
	// RenderFramedBox instead of RenderFramedBoxHeight to avoid double-padding
	content := m.List.VisibleContent(frame.DesiredContentLines)

	framed := ui.RenderFramedBox(m.title, header, content, footer, frame.FrameWidth)

	if m.confirmDialog.Visible {
		framed = ui.OverlayCentered(framed, m.confirmDialog.View(), frame.FrameWidth, frame.FrameHeight)
	}

	return framed
}
