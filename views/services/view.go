package servicesview

import (
	"fmt"
	"strings"
	"swarmcli/ui"
	filterlist "swarmcli/ui/components/filterable/list"
)

func (m *Model) View() string {
	width := m.List.Viewport.Width
	if width <= 0 {
		width = 80
	}

	// Add 4 to make frame full terminal width (app reduces viewport by 4 in normal mode)
	frameWidth := width + 4

	// Compute proportional column widths used by setRenderItem (6 columns)
	cols := 6
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

	labels := []string{" SERVICE", "STACK", "REPLICAS", "STATUS", "CREATED", "UPDATED"}
	header := ui.RenderColumnHeader(labels, colWidths)

	// Footer: cursor + optional search query
	status := fmt.Sprintf("Node %d of %d", m.List.Cursor+1, len(m.List.Filtered))
	statusBar := ui.StatusBarStyle.Render(status)

	var footer string
	if m.List.Mode == filterlist.ModeSearching {
		footer = ui.StatusBarStyle.Render("Filter (type then Enter): " + m.List.Query)
	} else if m.List.Query != "" {
		footer = ui.StatusBarStyle.Render("Filter: " + m.List.Query)
	}

	// Compute header/footer line counts
	headerLines := 0
	if header != "" {
		headerLines = 1
	}
	footerLines := 0
	if footer != "" {
		footerLines = len(strings.Split(footer, "\n"))
	}

	// Compose footer (status bar + optional filter line)
	if footer != "" {
		footer = statusBar + "\n" + footer
	} else {
		footer = statusBar
	}

	// Use the FilterableList's View() which already formats items and sets
	// the internal viewport content. Then pad/trim the textual lines to the
	// desired inner content height before framing (same approach as configs).
	content := m.List.View()

	// Reserve two lines from the viewport height for surrounding UI (helpbar/systeminfo)
	frameHeight := m.List.Viewport.Height - 2
	if frameHeight <= 0 {
		if m.height > 0 {
			frameHeight = m.height - 4
		}
		if frameHeight <= 0 {
			frameHeight = 20
		}
	}

	desiredContentLines := frameHeight - 2 - headerLines - footerLines
	if desiredContentLines < 0 {
		desiredContentLines = 0
	}

	contentLines := strings.Split(content, "\n")
	// Trim trailing empty lines
	for len(contentLines) > 0 && contentLines[len(contentLines)-1] == "" {
		contentLines = contentLines[:len(contentLines)-1]
	}
	// Pad or trim to desired length
	if len(contentLines) < desiredContentLines {
		for i := 0; i < desiredContentLines-len(contentLines); i++ {
			contentLines = append(contentLines, "")
		}
	} else if len(contentLines) > desiredContentLines {
		contentLines = contentLines[:desiredContentLines]
	}
	paddedContent := strings.Join(contentLines, "\n")

	framed := ui.RenderFramedBoxHeight(m.title, header, paddedContent, footer, frameWidth, frameHeight)

	// No debug overlays in final rendering

	if m.confirmDialog.Visible {
		framed = ui.OverlayCentered(framed, m.confirmDialog.View(), frameWidth, m.List.Viewport.Height)
	}
	if m.loading.Visible() {
		framed = ui.OverlayCentered(framed, m.loading.View(), frameWidth, m.List.Viewport.Height)
	}

	return framed
}
