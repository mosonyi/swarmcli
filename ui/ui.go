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

func OverlayCentered(base, overlay string, width, height int) string {
	baseLines := strings.Split(base, "\n")
	canvasHeight := len(baseLines)
	canvas := make([]string, canvasHeight)
	copy(canvas, baseLines)

	overlayLines := strings.Split(overlay, "\n")
	dialogHeight := len(overlayLines)
	if dialogHeight == 0 {
		return base
	}

	// Compute overlay width (visible width)
	dialogWidth := 0
	for _, l := range overlayLines {
		if w := lipgloss.Width(l); w > dialogWidth {
			dialogWidth = w
		}
	}

	// Center vertically (skip top/bottom border)
	startRow := (canvasHeight - dialogHeight) / 2
	if startRow < 1 {
		startRow = 1
	}
	if startRow+dialogHeight > canvasHeight-1 {
		startRow = canvasHeight - dialogHeight - 1
	}

	// Center horizontally inside frame (subtract 2 for borders)
	innerWidth := width - 2
	startCol := (innerWidth - dialogWidth) / 2
	if startCol < 0 {
		startCol = 0
	}

	for i, line := range overlayLines {
		row := startRow + i
		if row <= 0 || row >= canvasHeight-1 {
			continue
		}

		baseLine := canvas[row]
		if lipgloss.Width(baseLine) < 2 {
			continue
		}

		// Left/right borders
		leftBorder := string([]rune(baseLine)[0])
		rightBorder := string([]rune(baseLine)[len([]rune(baseLine))-1])

		// Fill inner area: pad left, overlay content, pad right
		rightPad := innerWidth - startCol - lipgloss.Width(line)
		if rightPad < 0 {
			rightPad = 0
		}

		inner := strings.Repeat(" ", startCol) + line + strings.Repeat(" ", rightPad)

		// Clamp inner to innerWidth exactly
		if lipgloss.Width(inner) > innerWidth {
			inner = lipgloss.NewStyle().MaxWidth(innerWidth).Render(inner)
		} else if lipgloss.Width(inner) < innerWidth {
			inner += strings.Repeat(" ", innerWidth-lipgloss.Width(inner))
		}

		canvas[row] = leftBorder + inner + rightBorder
	}

	return strings.Join(canvas, "\n")
}
