package contexts

import (
	"fmt"
	"path/filepath"
	"strings"
	"swarmcli/ui"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}

	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}

	title := "Docker Contexts"
	header := "Select a context to switch"

	if m.IsLoading() {
		header = "Loading contexts..."
	} else if m.IsSwitchPending() {
		header = "Switching context..."
	} else if err := m.GetError(); err != "" {
		header = fmt.Sprintf("Error: %s", err)
	} else if msg := m.GetSuccess(); msg != "" {
		header = msg
	}

	headerRendered := ui.FrameHeaderStyle.Render(header)

	// Build content
	var content strings.Builder
	contexts := m.GetContexts()
	cursor := m.GetCursor()

	if len(contexts) == 0 && !m.IsLoading() {
		content.WriteString("No Docker contexts found\n")
		content.WriteString("\n")
		content.WriteString("Press 'r' to refresh")
	} else {
		// Table header
		headerLine := fmt.Sprintf("%-4s %-20s %-45s %s", "CURR", "NAME", "DESCRIPTION", "DOCKER ENDPOINT")
		content.WriteString(lipgloss.NewStyle().Bold(true).Render(headerLine))
		content.WriteString("\n")

		// Contexts list
		for i, ctx := range contexts {
			current := " "
			if ctx.Current {
				current = "*"
			}

			// Truncate long values
			name := ctx.Name
			if len(name) > 18 {
				name = name[:15] + "..."
			}

			desc := ctx.Description
			if len(desc) > 43 {
				desc = desc[:40] + "..."
			}

			host := ctx.DockerHost
			if len(host) > 40 {
				host = host[:37] + "..."
			}

			line := fmt.Sprintf("%-4s %-20s %-45s %s", current, name, desc, host)

			// Highlight selected row
			if i == cursor {
				line = lipgloss.NewStyle().
					Background(lipgloss.Color("63")).
					Foreground(lipgloss.Color("230")).
					Render(line)
			}

			content.WriteString(line)
			content.WriteString("\n")
		}
	}

	frameWidth := width + 4
	height := m.viewport.Height

	// Pad content to fill viewport height
	contentLines := strings.Split(content.String(), "\n")
	// Account for frame borders (2), title (1), header (1) = 4 lines overhead
	availableLines := height - 4
	if availableLines < 0 {
		availableLines = 0
	}
	for len(contentLines) < availableLines {
		contentLines = append(contentLines, "")
	}
	paddedContent := strings.Join(contentLines, "\n")

	// Overlay dialogs on content BEFORE framing
	if m.fileBrowserActive {
		fileBrowserDialog := m.renderFileBrowserDialog()
		paddedContent = m.overlayDialog(paddedContent, fileBrowserDialog, width)
	} else if m.importInputActive {
		importDialog := m.renderImportDialog()
		paddedContent = m.overlayDialog(paddedContent, importDialog, width)
	} else if m.confirmDialog.Visible {
		dialogView := m.renderConfirmDialog()
		paddedContent = m.overlayDialog(paddedContent, dialogView, width)
	}

	rendered := ui.RenderFramedBox(
		title,
		headerRendered,
		paddedContent,
		"",
		frameWidth,
	)

	return rendered
}

// overlayDialog overlays a dialog on the content, centering it
func (m *Model) overlayDialog(content, dialog string, width int) string {
	contentLines := strings.Split(content, "\n")
	dialogLines := strings.Split(dialog, "\n")

	dialogHeight := len(dialogLines)
	dialogWidth := 0
	for _, line := range dialogLines {
		if w := lipgloss.Width(line); w > dialogWidth {
			dialogWidth = w
		}
	}

	// Center vertically
	startRow := (len(contentLines) - dialogHeight) / 2
	if startRow < 0 {
		startRow = 0
	}

	// Center horizontally
	startCol := (width - dialogWidth) / 2
	if startCol < 0 {
		startCol = 0
	}

	// Overlay dialog lines
	for i, dialogLine := range dialogLines {
		row := startRow + i
		if row < 0 || row >= len(contentLines) {
			continue
		}

		baseLine := contentLines[row]
		baseWidth := lipgloss.Width(baseLine)

		// Build new line with dialog centered
		var newLine strings.Builder

		if baseWidth < startCol {
			// Base line is shorter than where dialog should start
			newLine.WriteString(baseLine)
			newLine.WriteString(strings.Repeat(" ", startCol-baseWidth))
			newLine.WriteString(dialogLine)
		} else {
			// Overlay dialog in the middle
			leftPart := ""
			rightPart := ""

			// Get left part (up to startCol)
			if startCol > 0 {
				leftPart = baseLine[:min(startCol, len(baseLine))]
			}

			// Get right part (after dialog)
			rightStart := startCol + dialogWidth
			if rightStart < baseWidth && rightStart < len(baseLine) {
				rightPart = baseLine[rightStart:]
			}

			newLine.WriteString(leftPart)
			newLine.WriteString(dialogLine)
			newLine.WriteString(rightPart)
		}

		contentLines[row] = newLine.String()
	}

	return strings.Join(contentLines, "\n")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (m *Model) renderImportDialog() string {
	contentWidth := 60

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("63")).
		Padding(0, 1).
		Width(contentWidth)

	itemStyle := lipgloss.NewStyle().
		Padding(0, 1).
		Width(contentWidth)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("117")).
		Width(contentWidth + 2)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 1).
		Width(contentWidth)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("63")).
		Bold(true)

	var lines []string
	lines = append(lines, titleStyle.Render(" Import Docker Context "))
	lines = append(lines, itemStyle.Render("Enter the path to the context tar file:"))
	lines = append(lines, itemStyle.Render(""))
	lines = append(lines, itemStyle.Render(m.importInput.View()))
	lines = append(lines, itemStyle.Render(""))

	helpText := fmt.Sprintf(" %s Import • %s Cancel",
		keyStyle.Render("<Enter>"),
		keyStyle.Render("<Esc>"))
	lines = append(lines, helpStyle.Render(helpText))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return borderStyle.Render(content)
}

// renderConfirmDialog renders the confirmation dialog
func (m *Model) renderConfirmDialog() string {
	contentWidth := 60

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("63")).
		Padding(0, 1)

	itemStyle := lipgloss.NewStyle().
		Padding(0, 1)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("117"))

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("63")).
		Bold(true)

	// Helper function to ensure exact width
	ensureWidth := func(s string, width int) string {
		currentWidth := lipgloss.Width(s)
		if currentWidth < width {
			return s + strings.Repeat(" ", width-currentWidth)
		}
		return s
	}

	var lines []string
	lines = append(lines, ensureWidth(titleStyle.Render(" Confirmation "), contentWidth))
	lines = append(lines, ensureWidth(itemStyle.Render(""), contentWidth))
	lines = append(lines, ensureWidth(itemStyle.Render(m.confirmDialog.Message), contentWidth))
	lines = append(lines, ensureWidth(itemStyle.Render(""), contentWidth))

	helpText := fmt.Sprintf(" %s Yes • %s No",
		keyStyle.Render("<y>"),
		keyStyle.Render("<n>"))
	lines = append(lines, ensureWidth(helpStyle.Render(helpText), contentWidth))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return borderStyle.Render(content)
}

// renderFileBrowserDialog renders the file browser dialog
func (m *Model) renderFileBrowserDialog() string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("63")).
		Padding(0, 1)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("117"))

	itemStyle := lipgloss.NewStyle().
		Padding(0, 1)

	selectedStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("63")).
		Padding(0, 1)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("63")).
		Bold(true)

	var lines []string
	lines = append(lines, titleStyle.Render(fmt.Sprintf(" Select .tar file (%d files) ", len(m.fileBrowserFiles))))
	lines = append(lines, itemStyle.Render(""))

	// Show files with cursor
	maxVisible := 10
	start := m.fileBrowserCursor - maxVisible/2
	if start < 0 {
		start = 0
	}
	end := start + maxVisible
	if end > len(m.fileBrowserFiles) {
		end = len(m.fileBrowserFiles)
		start = end - maxVisible
		if start < 0 {
			start = 0
		}
	}

	for i := start; i < end; i++ {
		filePath := m.fileBrowserFiles[i]
		fileName := filepath.Base(filePath)
		if i == m.fileBrowserCursor {
			lines = append(lines, selectedStyle.Render("→ "+fileName))
		} else {
			lines = append(lines, itemStyle.Render("  "+fileName))
		}
	}

	lines = append(lines, itemStyle.Render(""))
	helpText := fmt.Sprintf(" %s Select • %s Navigate • %s Cancel",
		keyStyle.Render("<Enter>"),
		keyStyle.Render("<↑/↓>"),
		keyStyle.Render("<Esc>"))
	lines = append(lines, helpStyle.Render(helpText))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return borderStyle.Render(content)
}
