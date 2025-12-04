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

	// Center vertically in the entire canvas
	startRow := (canvasHeight - dialogHeight) / 2
	if startRow < 0 {
		startRow = 0
	}
	if startRow+dialogHeight > canvasHeight {
		startRow = canvasHeight - dialogHeight
		if startRow < 0 {
			startRow = 0
		}
	}

	// Center horizontally - calculate based on the actual canvas width
	// For framed content, we need to account for borders (subtract 2)
	// For fullscreen, use the full width
	canvasWidth := 0
	if len(baseLines) > 0 {
		canvasWidth = lipgloss.Width(baseLines[0])
	}
	if canvasWidth == 0 {
		canvasWidth = width
	}

	// Check if this is a framed box (has borders)
	hasFrameBorders := len(baseLines) > 0 && (strings.HasPrefix(baseLines[0], "╭") || strings.HasPrefix(baseLines[0], "│"))

	innerWidth := canvasWidth
	if hasFrameBorders {
		innerWidth = canvasWidth - 2 // subtract left and right borders
	}

	startCol := (innerWidth - dialogWidth) / 2
	if startCol < 0 {
		startCol = 0
	}

	for i, line := range overlayLines {
		row := startRow + i
		if row < 0 || row >= canvasHeight {
			continue
		}

		baseLine := canvas[row]
		baseWidth := lipgloss.Width(baseLine)
		if baseWidth < 2 {
			continue
		}

		// Get the actual width of the overlay line
		overlayLineWidth := lipgloss.Width(line)

		// For framed boxes, preserve borders and blank out the content area
		if hasFrameBorders {
			// Simple and robust approach: find first and last pipe character
			firstPipe := strings.Index(baseLine, "│")
			lastPipe := strings.LastIndex(baseLine, "│")

			// If we can't find proper borders, skip this line (shouldn't happen in a well-formed frame)
			if firstPipe == -1 || lastPipe == -1 || firstPipe >= lastPipe {
				continue
			}

			// Extract parts: everything before first pipe, content between pipes, everything after last pipe
			leftPart := baseLine[:firstPipe+len("│")]
			rightPart := baseLine[lastPipe:]
			middleContent := baseLine[firstPipe+len("│") : lastPipe]

			// Calculate visual width of middle content (accounts for ANSI codes)
			middleVisualWidth := lipgloss.Width(middleContent)

			// Ensure we have reasonable dimensions
			if middleVisualWidth <= 0 {
				continue
			}

			// Completely blank the middle area for ALL lines in the dialog range
			// and only draw the dialog content on the appropriate lines
			leftPadSize := (middleVisualWidth - overlayLineWidth) / 2
			if leftPadSize < 0 {
				leftPadSize = 0
			}

			rightPadSize := middleVisualWidth - leftPadSize - overlayLineWidth
			if rightPadSize < 0 {
				rightPadSize = 0
			}

			// Build completely blank line with centered dialog content
			newMiddle := strings.Repeat(" ", leftPadSize) + line + strings.Repeat(" ", rightPadSize)

			// Pad to exact width if needed
			actualWidth := lipgloss.Width(newMiddle)
			if actualWidth < middleVisualWidth {
				newMiddle += strings.Repeat(" ", middleVisualWidth-actualWidth)
			} else if actualWidth > middleVisualWidth {
				// Truncate if somehow too wide
				excess := actualWidth - middleVisualWidth
				if excess <= rightPadSize {
					rightPadSize -= excess
					if rightPadSize < 0 {
						rightPadSize = 0
					}
					newMiddle = strings.Repeat(" ", leftPadSize) + line + strings.Repeat(" ", rightPadSize)
				}
			}

			canvas[row] = leftPart + newMiddle + rightPart
		} else {
			// Fullscreen mode
			baseRunes := []rune(baseLine)

			leftContent := ""
			if startCol > 0 && startCol <= len(baseRunes) {
				leftContent = string(baseRunes[:startCol])
			} else if startCol > len(baseRunes) {
				leftContent = string(baseRunes)
				leftContent += strings.Repeat(" ", startCol-len(baseRunes))
			}

			rightContent := ""
			afterOverlay := startCol + overlayLineWidth
			if afterOverlay < len(baseRunes) {
				rightContent = string(baseRunes[afterOverlay:])
			}

			// Pad if needed to maintain width
			combined := leftContent + line + rightContent
			combinedWidth := lipgloss.Width(combined)
			if combinedWidth < canvasWidth {
				combined += strings.Repeat(" ", canvasWidth-combinedWidth)
			}

			canvas[row] = combined
		}
	}

	return strings.Join(canvas, "\n")
}
