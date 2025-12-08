package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Styles (you can override these per-view if desired)
var (
	FrameTitleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("81")).
			Bold(true)

	FrameHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("75")).
				Bold(true)

	FrameBorderColor = lipgloss.Color("117")
)

// RenderFramedBox draws a bordered frame with title, optional header, and content.
// If width <= 0, defaults to content width + padding.
// ANSI sequences in content are preserved.
func RenderFramedBox(title, header, content, footer string, width int) string {
	lines := strings.Split(content, "\n")
	footerLines := []string{}
	if footer != "" {
		footerLines = strings.Split(footer, "\n")
	}

	// Compute content width
	contentWidth := 0
	for _, l := range append(lines, footerLines...) {
		if w := lipgloss.Width(l); w > contentWidth {
			contentWidth = w
		}
	}
	if width <= 0 {
		width = contentWidth + 4 // padding left/right
	}

	titleStyled := FrameTitleStyle.Render(" " + title + " ")
	headerStyled := FrameHeaderStyle.Render(header)

	borderWidth := width - 2 // left/right borders

	// Border style
	borderStyle := lipgloss.NewStyle().Foreground(FrameBorderColor)

	// Top border with centered title
	leftPad := (borderWidth - lipgloss.Width(titleStyled)) / 2
	if leftPad < 0 {
		leftPad = 0
	}
	rightPad := borderWidth - leftPad - lipgloss.Width(titleStyled)
	if rightPad < 0 {
		rightPad = 0
	}

	topLine := fmt.Sprintf(
		"%s%s%s%s%s",
		borderStyle.Render("╭"),
		borderStyle.Render(strings.Repeat("─", leftPad)),
		titleStyled,
		borderStyle.Render(strings.Repeat("─", rightPad)),
		borderStyle.Render("╮"),
	)

	// Box lines start with top border
	boxLines := []string{topLine}

	// Optional header
	if header != "" {
		boxLines = append(boxLines, fmt.Sprintf("%s%s%s",
			borderStyle.Render("│"),
			padLine(headerStyled, borderWidth),
			borderStyle.Render("│")))
	}

	// Content
	for _, l := range lines {
		boxLines = append(boxLines, fmt.Sprintf("%s%s%s",
			borderStyle.Render("│"),
			padLine(l, borderWidth),
			borderStyle.Render("│")))
	}

	// Optional footer (above bottom border)
	for _, fl := range footerLines {
		boxLines = append(boxLines, fmt.Sprintf("%s%s%s",
			borderStyle.Render("│"),
			padLine(fl, borderWidth),
			borderStyle.Render("│")))
	}

	// Bottom border
	bottomLine := fmt.Sprintf("%s%s%s",
		borderStyle.Render("╰"),
		borderStyle.Render(strings.Repeat("─", borderWidth)),
		borderStyle.Render("╯"))
	boxLines = append(boxLines, bottomLine)

	return strings.Join(boxLines, "\n")
}

// padLine fits a line to width, preserving ANSI sequences
func padLine(line string, width int) string {
	l := lipgloss.Width(line)
	if l >= width {
		// Truncate but ensure we leave room for proper ending if needed
		// Use MaxWidth to handle ANSI sequences properly
		truncated := lipgloss.NewStyle().MaxWidth(width).Render(line)
		// Ensure the truncated line is exactly the visual width requested
		truncatedWidth := lipgloss.Width(truncated)
		if truncatedWidth < width {
			truncated += strings.Repeat(" ", width-truncatedWidth)
		}
		return truncated
	}
	return line + strings.Repeat(" ", width-l)
}

// RenderConfirmDialog renders a standard confirmation dialog with y/n options
func RenderConfirmDialog(message string) string {
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
	lines = append(lines, ensureWidth(itemStyle.Render(message), contentWidth))
	lines = append(lines, ensureWidth(itemStyle.Render(""), contentWidth))

	helpText := fmt.Sprintf(" %s Yes • %s No",
		keyStyle.Render("<y>"),
		keyStyle.Render("<n>"))
	lines = append(lines, ensureWidth(helpStyle.Render(helpText), contentWidth))

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return borderStyle.Render(content)
}

func OverlayCentered(base, overlay string, width, height int) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")

	dialogHeight := len(overlayLines)
	dialogWidth := 0
	for _, line := range overlayLines {
		if w := lipgloss.Width(line); w > dialogWidth {
			dialogWidth = w
		}
	}

	// Center vertically
	startRow := (len(baseLines) - dialogHeight) / 2
	if startRow < 0 {
		startRow = 0
	}

	// Center horizontally
	startCol := (width - dialogWidth) / 2
	if startCol < 0 {
		startCol = 0
	}

	// Overlay dialog lines
	for i, dialogLine := range overlayLines {
		row := startRow + i
		if row < 0 || row >= len(baseLines) {
			continue
		}

		baseLine := baseLines[row]
		baseWidth := lipgloss.Width(baseLine)

		// Build new line with dialog centered
		var newLine strings.Builder

		if baseWidth < startCol {
			// Base line is shorter than where dialog should start
			newLine.WriteString(baseLine)
			newLine.WriteString(strings.Repeat(" ", startCol-baseWidth))
			newLine.WriteString(dialogLine)
		} else {
			// Overlay dialog in the middle using width-aware truncation
			leftPart := truncateANSI(baseLine, startCol)
			rightStart := startCol + dialogWidth
			rightPart := ""
			if rightStart < baseWidth {
				// Skip the overlay width and get the rest
				rightPart = truncateANSIAfter(baseLine, rightStart)
			}

			newLine.WriteString(leftPart)
			newLine.WriteString(dialogLine)
			newLine.WriteString(rightPart)
		}

		baseLines[row] = newLine.String()
	}

	return strings.Join(baseLines, "\n")
}

// truncateANSI truncates a string with ANSI codes to a specific visual width
func truncateANSI(s string, width int) string {
	if width <= 0 {
		return ""
	}
	var result strings.Builder
	var currentWidth int
	inEscape := false
	
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
		}
		
		if inEscape {
			result.WriteRune(r)
			if r == 'm' {
				inEscape = false
			}
			continue
		}
		
		if currentWidth >= width {
			break
		}
		
		result.WriteRune(r)
		currentWidth++
	}
	
	return result.String()
}

// truncateANSIAfter skips characters up to a width and returns the rest
func truncateANSIAfter(s string, skipWidth int) string {
	if skipWidth <= 0 {
		return s
	}
	var result strings.Builder
	var currentWidth int
	inEscape := false
	var escapeBuffer strings.Builder
	
	for _, r := range s {
		if r == '\x1b' {
			inEscape = true
			escapeBuffer.Reset()
		}
		
		if inEscape {
			escapeBuffer.WriteRune(r)
			if r == 'm' {
				inEscape = false
				if currentWidth >= skipWidth {
					result.WriteString(escapeBuffer.String())
				}
			}
			continue
		}
		
		if currentWidth >= skipWidth {
			result.WriteRune(r)
		}
		currentWidth++
	}
	
	return result.String()
}
