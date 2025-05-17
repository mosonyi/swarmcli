package styles

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
	"strings"
)

var (
	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(0, 1)

	StatusStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00FF00")).
			Padding(0, 1).
			Width(50)

	ListStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#FFD700")).
			Margin(1, 0).
			Padding(1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true).
			Margin(1, 0)
)

func Frame(title, content string, width int) string {
	const (
		horizontal  = "─"
		vertical    = "│"
		topLeft     = "┌"
		topRight    = "┐"
		bottomLeft  = "└"
		bottomRight = "┘"
		edgePadding = 4 // 2 for vertical borders, 2 for spacing
	)

	lines := strings.Split(content, "\n")

	// Ensure minimum width for the frame
	minWidth := len(title) + edgePadding
	if width < minWidth {
		width = minWidth
	}

	// Top border with title
	titleSpace := width - len(title) - 2
	top := fmt.Sprintf("%s%s%s%s\n",
		topLeft,
		strings.Repeat(horizontal, titleSpace/2),
		title,
		strings.Repeat(horizontal, titleSpace-titleSpace/2)+topRight,
	)

	// Frame content
	var body strings.Builder
	for _, line := range lines {
		truncated := line
		if len(truncated) > width-2 {
			truncated = truncated[:width-2]
		}
		padding := width - 2 - len(truncated)
		body.WriteString(fmt.Sprintf("%s%s%s%s\n", vertical, truncated, strings.Repeat(" ", padding), vertical))
	}

	// Bottom border
	bottom := fmt.Sprintf("%s%s%s", bottomLeft, strings.Repeat(horizontal, width-2), bottomRight)

	return top + body.String() + bottom
}
