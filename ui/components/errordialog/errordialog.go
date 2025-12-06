package errordialog

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Render renders an error dialog with the given error message
func Render(errorMsg string) string {
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("196")).
		Padding(0, 1)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("196"))

	itemStyle := lipgloss.NewStyle().
		Padding(0, 1)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 1)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("63")).
		Bold(true)

	var lines []string
	lines = append(lines, titleStyle.Render(" Error "))
	lines = append(lines, itemStyle.Render(""))

	maxWidth := 70
	wrappedLines := wrapText(errorMsg, maxWidth)
	for _, line := range wrappedLines {
		lines = append(lines, itemStyle.Render(line))
	}

	lines = append(lines, itemStyle.Render(""))
	helpText := fmt.Sprintf("%s %s %s",
		helpStyle.Render("Press"),
		keyStyle.Render("<Enter>"),
		helpStyle.Render("to close"))
	lines = append(lines, helpText)

	content := lipgloss.JoinVertical(lipgloss.Left, lines...)
	return borderStyle.Render(content)
}

func wrapText(text string, width int) []string {
	if len(text) <= width {
		return []string{text}
	}

	var lines []string
	words := strings.Fields(text)
	currentLine := ""

	for _, word := range words {
		if len(currentLine)+len(word)+1 <= width {
			if currentLine == "" {
				currentLine = word
			} else {
				currentLine += " " + word
			}
		} else {
			if currentLine != "" {
				lines = append(lines, currentLine)
			}
			currentLine = word
		}
	}

	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}
