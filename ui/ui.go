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

// RenderFramedBox draws a bordered frame with title, optional header, and content.
// If width <= 0, defaults to content width + padding.
// ANSI sequences in content are preserved.
func RenderFramedBox(title, header, content string, width int) string {
	lines := strings.Split(content, "\n")

	// Compute content width
	contentWidth := 0
	for _, l := range lines {
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
		"╭%s%s%s╮",
		strings.Repeat("─", leftPad),
		titleStyled,
		strings.Repeat("─", rightPad),
	)

	// Header line
	boxLines := []string{topLine}
	if header != "" {
		boxLines = append(boxLines, fmt.Sprintf("│%s│", padLine(headerStyled, borderWidth)))
	}

	// Content lines
	for _, l := range lines {
		boxLines = append(boxLines, fmt.Sprintf("│%s│", padLine(l, borderWidth)))
	}

	// Bottom border
	bottomLine := fmt.Sprintf("╰%s╯", strings.Repeat("─", borderWidth))
	boxLines = append(boxLines, bottomLine)

	return strings.Join(boxLines, "\n")
}

// padLine fits a line to width, preserving ANSI sequences
func padLine(line string, width int) string {
	l := lipgloss.Width(line)
	if l >= width {
		return lipgloss.NewStyle().MaxWidth(width).Render(line)
	}
	return line + strings.Repeat(" ", width-l)
}

// overlayCentered overlays `overlay` on top of `base` content, centered.
// Base content remains fully visible; overlay is drawn on top.
func OverlayCentered(base, overlay string, width, height int) string {
	baseLines := strings.Split(base, "\n")
	canvasHeight := max(len(baseLines), height)
	canvas := make([]string, canvasHeight)

	// Copy base lines
	for i := 0; i < canvasHeight; i++ {
		if i < len(baseLines) {
			canvas[i] = padRight(baseLines[i], width)
		} else {
			canvas[i] = strings.Repeat(" ", width)
		}
	}

	overlayLines := strings.Split(overlay, "\n")
	overlayHeight := len(overlayLines)
	overlayWidth := 0
	for _, l := range overlayLines {
		if w := lipgloss.Width(l); w > overlayWidth {
			overlayWidth = w
		}
	}

	startRow := (canvasHeight - overlayHeight) / 2
	if startRow < 0 {
		startRow = 0
	}
	startCol := (width - overlayWidth) / 2
	if startCol < 0 {
		startCol = 0
	}

	// Draw overlay
	for i, line := range overlayLines {
		row := startRow + i
		if row >= len(canvas) {
			break
		}
		prefix := strings.Repeat(" ", startCol)
		suffix := ""
		if startCol+lipgloss.Width(line) < width {
			suffix = strings.Repeat(" ", width-startCol-lipgloss.Width(line))
		}
		canvas[row] = prefix + line + suffix
	}

	return strings.Join(canvas, "\n")
}

func padRight(s string, width int) string {
	l := lipgloss.Width(s)
	if l >= width {
		return s
	}
	return s + strings.Repeat(" ", width-l)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
