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
			// Find the BORDER pipes, not content pipes
			// The left border pipe should be at the start (possibly after ANSI codes)
			// The right border pipe should be at the end (possibly before ANSI codes)

			// Find first non-ANSI character position and check if it's a pipe
			firstPipe := -1
			lastPipe := -1

			// Scan from start to find left border (skip ANSI codes)
			inAnsi := false
			for i := 0; i < len(baseLine); i++ {
				if baseLine[i] == '\x1b' && i+1 < len(baseLine) && baseLine[i+1] == '[' {
					inAnsi = true
					continue
				}
				if inAnsi {
					if baseLine[i] == 'm' {
						inAnsi = false
					}
					continue
				}
				// Found first non-ANSI character
				if strings.HasPrefix(baseLine[i:], "│") {
					firstPipe = i
				}
				break
			}

			// Scan from end to find right border (skip ANSI codes backward)
			for i := len(baseLine) - 1; i >= 0; i-- {
				if strings.HasPrefix(baseLine[i:], "│") {
					// Check if this is likely the right border by seeing if there's only ANSI codes after it
					afterPipe := baseLine[i+len("│"):]
					// Should only contain ANSI reset codes or be empty
					if len(afterPipe) == 0 || strings.HasPrefix(afterPipe, "\x1b[") {
						lastPipe = i
						break
					}
				}
			}

			// If we can't find proper borders, skip this line
			if firstPipe == -1 || lastPipe == -1 || firstPipe >= lastPipe {
				continue
			}

			// Validate that we have reasonable positions
			baseLineVisualWidth := lipgloss.Width(baseLine)
			if baseLineVisualWidth < 2 {
				continue
			}

			// Extract the left part INCLUDING the left border (preserves original ANSI codes)
			leftPart := baseLine[:firstPipe+len("│")]
			// Extract the middle content between the two pipes
			middleContent := baseLine[firstPipe+len("│") : lastPipe]

			// Calculate visual width of middle content (accounts for ANSI codes)
			middleVisualWidth := lipgloss.Width(middleContent)

			// Ensure we have reasonable dimensions
			if middleVisualWidth <= 0 {
				continue
			}

			// The expected total width should be: leftBorder(1) + middleWidth + rightBorder(1)
			// If the calculated middle is much different than expected, something is wrong
			expectedMiddleWidth := baseLineVisualWidth - 2
			if middleVisualWidth < expectedMiddleWidth-5 || middleVisualWidth > expectedMiddleWidth+5 {
				// Width mismatch - likely ANSI code issue, use expected width
				middleVisualWidth = expectedMiddleWidth
				if middleVisualWidth <= 0 {
					continue
				}
			}

			// Center the dialog in the middle content area
			// RenderFramedBox adds +4 to width (line 41 of this file), so borderWidth has +2 padding
			// The padding is added at the END by padLine (line 119), so we need to strip it first

			// Strip trailing padding spaces from middleContent to get ACTUAL content
			middleStr := strings.TrimRight(middleContent, " ")
			actualContentLen := len(middleStr)

			// Calculate dialog position based on the ORIGINAL content width (minus 2 padding)
			originalContentWidth := middleVisualWidth - 2
			dialogStartPos := (originalContentWidth - overlayLineWidth) / 2
			if dialogStartPos < 0 {
				dialogStartPos = 0
			}

			// Build new middle line by blanking out the dialog area
			var newMiddleBuilder strings.Builder

			// Take left part (before dialog) from the ACTUAL content
			if dialogStartPos > 0 {
				if dialogStartPos <= actualContentLen {
					newMiddleBuilder.WriteString(middleStr[:dialogStartPos])
				} else {
					newMiddleBuilder.WriteString(middleStr)
					newMiddleBuilder.WriteString(strings.Repeat(" ", dialogStartPos-actualContentLen))
				}
			}

			// Add the dialog line (which will hide content behind it)
			newMiddleBuilder.WriteString(line)

			// Add right part (after dialog) from the ACTUAL content
			dialogEndPos := dialogStartPos + overlayLineWidth
			if dialogEndPos < actualContentLen {
				newMiddleBuilder.WriteString(middleStr[dialogEndPos:])
			}

			// Pad to full width (restore the padding to match middleVisualWidth)
			newMiddle := newMiddleBuilder.String()
			if len(newMiddle) < middleVisualWidth {
				newMiddle += strings.Repeat(" ", middleVisualWidth-len(newMiddle))
			}

			// Reconstruct: preserve left part with original ANSI, new middle, fresh right border
			borderStyle := lipgloss.NewStyle().Foreground(FrameBorderColor)
			canvas[row] = leftPart + newMiddle + borderStyle.Render("│")
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
