package helpview

import (
	"fmt"
	"strings"
	"swarmcli/ui"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}

	// Use categorized view if categories are provided
	if len(m.categories) > 0 {
		return m.renderCategorizedHelp()
	}

	// Legacy simple help view
	header := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("#00d7ff")).
		Render("Available Commands")

	body := m.content
	footer := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#808080")).
		Render("[press q or esc to go back]")

	return ui.BorderStyle.Render(
		fmt.Sprintf("%s\n\n%s\n\n%s", header, body, footer),
	)
}

func (m *Model) renderCategorizedHelp() string {
	width := m.Viewable.Width
	if width <= 0 {
		width = 80
	}

	// Styles - use ANSI 256-color codes (k9s uses "green" = color 2)
	categoryStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("2")) // Standard green

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("33")). // Dodger blue
		Bold(true)

	descStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("252")) // Light gray

	// Calculate column widths
	numCols := len(m.categories)
	if numCols == 0 {
		numCols = 1
	}
	colWidth := width / numCols
	maxKeyWidth := 15 // Fixed width for keys

	// Find max rows needed
	maxRows := 0
	for _, cat := range m.categories {
		if len(cat.Items) > maxRows {
			maxRows = len(cat.Items)
		}
	}

	// Build header row with category titles in GREEN
	var headerParts []string
	for _, cat := range m.categories {
		titleText := strings.ToUpper(cat.Title)
		// Pad to column width first, then apply style
		paddedTitle := fmt.Sprintf("%-*s", colWidth, titleText)
		styledTitle := categoryStyle.Render(paddedTitle)
		headerParts = append(headerParts, styledTitle)
	}
	headerRow := strings.Join(headerParts, "")

	// Build content rows
	var contentLines []string
	for row := 0; row < maxRows; row++ {
		var rowParts []string
		for _, cat := range m.categories {
			if row < len(cat.Items) {
				item := cat.Items[row]
				// Format key and description with proper alignment
				styledKey := keyStyle.Render(fmt.Sprintf("%-*s", maxKeyWidth, item.Keys))
				styledDesc := descStyle.Render(item.Description)

				// Combine and pad to column width
				// We need to account for visual width without ANSI codes for padding
				plainText := fmt.Sprintf("%-*s %s", maxKeyWidth, item.Keys, item.Description)
				if len(plainText) > colWidth {
					// Truncate description if too long
					descWidth := colWidth - maxKeyWidth - 1
					if descWidth > 0 && len(item.Description) > descWidth {
						styledDesc = descStyle.Render(item.Description[:descWidth-3] + "...")
						plainText = fmt.Sprintf("%-*s %s", maxKeyWidth, item.Keys, item.Description[:descWidth-3]+"...")
					}
				}

				// Now render styled version maintaining the same width
				styledLine := styledKey + " " + styledDesc
				// Add padding to match column width
				paddingNeeded := colWidth - len(plainText)
				if paddingNeeded > 0 {
					styledLine += strings.Repeat(" ", paddingNeeded)
				}

				rowParts = append(rowParts, styledLine)
			} else {
				// Empty cell
				rowParts = append(rowParts, strings.Repeat(" ", colWidth))
			}
		}
		contentLines = append(contentLines, strings.Join(rowParts, ""))
	}

	// Combine header and content
	fullContent := headerRow + "\n\n" + strings.Join(contentLines, "\n")

	// Use same pattern as inspect
	title := "Help"
	header := ""
	footer := ui.StatusBarStyle.Render("Press <esc> to go back")

	frame := ui.ComputeFrameDimensions(
		width,
		m.Viewable.Height,
		width,
		m.height,
		header,
		footer,
	)

	// Trim content to fit frame (preserving ANSI codes)
	viewportContent := ui.TrimOrPadContentToLines(fullContent, frame.DesiredContentLines)

	// Render framed box
	return ui.RenderFramedBox(title, header, viewportContent, footer, frame.FrameWidth)
}
