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

	FrameBorderColor = lipgloss.Color("240")
)

// RenderFramedBox draws a bordered frame with a centered title, header line, and content.
// It preserves ANSI ui and ensures the frame width is consistent.
func RenderFramedBox(title string, header string, content string, width int) string {
	if width <= 0 {
		width = 80
	}

	titleStyled := FrameTitleStyle.Render(" " + title + " ")
	headerStyled := FrameHeaderStyle.Render(header)

	// Build top border with centered title
	topBorderLeft := "╭"
	topBorderRight := "╮"
	borderWidth := width - lipgloss.Width(topBorderLeft+topBorderRight)
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
		topBorderLeft,
		strings.Repeat("─", leftPad),
		titleStyled,
		strings.Repeat("─", rightPad),
		topBorderRight,
	)

	// Build content area
	lines := []string{}
	if header != "" {
		lines = append(lines, fmt.Sprintf("│%s│", padLine(headerStyled, borderWidth)))
	}

	for _, line := range strings.Split(content, "\n") {
		lines = append(lines, fmt.Sprintf("│%s│", padLine(line, borderWidth)))
	}

	bottomLine := fmt.Sprintf("╰%s╯", strings.Repeat("─", borderWidth))

	return strings.Join(append([]string{topLine}, append(lines, bottomLine)...), "\n")
}

// padLine safely fits styled text to the given width (preserving ANSI sequences).
func padLine(line string, width int) string {
	lineWidth := lipgloss.Width(line)
	if lineWidth == width {
		return line
	}
	if lineWidth < width {
		return line + strings.Repeat(" ", width-lineWidth)
	}
	return lipgloss.NewStyle().MaxWidth(width).Render(line)
}
