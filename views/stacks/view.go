package stacksview

import (
	"fmt"
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

	content := m.List.View()

	// Add 4 to make frame full terminal width (app reduces viewport by 4 in normal mode)
	frameWidth := m.List.Viewport.Width + 4
	return ui.RenderFramedBox(title, header, content, footer, frameWidth)
}
