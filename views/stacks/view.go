package stacksview

import (
	"fmt"
	"strings"
	"swarmcli/ui"
	filterlist "swarmcli/ui/components/filterable/list"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}

	title := fmt.Sprintf("Stacks on Node (Total: %d)", len(m.List.Items))

	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15"))

	// Use the same column width as computed for items
	colWidth := m.List.GetColWidth()
	if colWidth < 15 {
		colWidth = 15
	}

	// Format: %-*s (stack name) + 8 spaces + left-aligned SERVICES
	header := headerStyle.Render(fmt.Sprintf("%-*s        %-s", colWidth, "STACK", "SERVICES"))

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

	// Use the FilterableList's View() which formats items and sets
	// the internal viewport content. This keeps behavior consistent
	// with `configs` and `services` and avoids layout mismatches.
	content := m.List.View()

	// Add 4 to make frame full terminal width (app reduces viewport by 4 in normal mode)
	frameWidth := m.List.Viewport.Width + 4

	// Compute frameHeight from viewport (treat Viewport.Height as the total
	// frame height like `configs` view does). Then compute desired inner
	// content lines = frameHeight - borders - header - footer, and pad/trim
	// content to that length.
	// Subtract 2 to account for the app's stackbar and bottom status line
	frameHeight := m.List.Viewport.Height - 2
	if frameHeight <= 0 {
		frameHeight = 20
	}

	headerLines := 0
	if header != "" {
		headerLines = len(strings.Split(header, "\n"))
	}
	footerLines := 0
	if footer != "" {
		footerLines = len(strings.Split(footer, "\n"))
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
	if len(contentLines) < desiredContentLines {
		for i := 0; i < desiredContentLines-len(contentLines); i++ {
			contentLines = append(contentLines, "")
		}
	} else if len(contentLines) > desiredContentLines {
		contentLines = contentLines[:desiredContentLines]
	}
	paddedContent := strings.Join(contentLines, "\n")

	framed := ui.RenderFramedBoxHeight(title, header, paddedContent, footer, frameWidth, frameHeight)

	// Final rendering (no debug overlay)
	return framed
}
