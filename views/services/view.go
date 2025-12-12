package servicesview

import (
	"fmt"
	"strings"
	"swarmcli/ui"
	filterlist "swarmcli/ui/components/filterable/list"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	width := m.List.Viewport.Width
	if width <= 0 {
		width = 80
	}

	// Add 4 to make frame full terminal width (app reduces viewport by 4 in normal mode)
	frameWidth := width + 4

	// Compute dynamic column widths (same as in setRenderItem)
	replicaWidth := 10
	statusWidth := 12
	createdWidth := 10
	updatedWidth := 10
	maxService := len("SERVICE")
	maxStack := len("STACK")
	for _, e := range m.List.Filtered {
		if len(e.ServiceName) > maxService {
			maxService = len(e.ServiceName)
		}
		if len(e.StackName) > maxStack {
			maxStack = len(e.StackName)
		}
	}
	total := maxService + maxStack + replicaWidth + statusWidth + createdWidth + updatedWidth + 40 // spacing between columns
	if total > width {
		overflow := total - width
		if maxStack > maxService {
			maxStack -= overflow
			if maxStack < 5 {
				maxStack = 5
			}
		} else {
			maxService -= overflow
			if maxService < 5 {
				maxService = 5
			}
		}
	}

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15"))

	header := headerStyle.Render(fmt.Sprintf(
		"%-*s        %-*s        %-*s        %-*s        %-*s        %-*s",
		maxService, "SERVICE",
		maxStack, "STACK",
		replicaWidth, "REPLICAS",
		statusWidth, "STATUS",
		createdWidth, "CREATED",
		updatedWidth, "UPDATED",
	))

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
