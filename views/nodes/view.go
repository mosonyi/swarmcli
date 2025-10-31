package nodesview

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("81")). // bluish
			Bold(true)

	headerStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("75")). // slightly bluish
			Bold(true)

	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("63")).
			Bold(true)

	statusBarStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("250")).
			Background(lipgloss.Color("237")).
			Padding(0, 1)
)

// View renders the node list with a full k9s-style border and integrated header title.
func (m Model) View() string {
	if !m.Visible {
		return ""
	}

	total := len(m.nodes)
	managers := 0
	for _, n := range m.nodes {
		if n.ManagerStatus != "" {
			managers++
		}
	}

	// Title text (plain)
	titlePlain := fmt.Sprintf(" Nodes (%d total, %d manager%s) ", total, managers, plural(managers))
	titleStyled := titleStyle.Render(titlePlain)

	// Header plain (we will style it but pad/truncate safely)
	headerPlain := "HOSTNAME              STATUS     AVAILABILITY   MANAGER STATUS"
	headerStyled := headerStyle.Render(headerPlain)

	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}

	// --- Build top border with centered title
	topBorderLeft := "╭"
	topBorderRight := "╮"

	borderWidth := width - lipgloss.Width(topBorderLeft+topBorderRight)
	if borderWidth < lipgloss.Width(titleStyled) {
		borderWidth = lipgloss.Width(titleStyled) + 2
	}

	leftPad := (borderWidth - lipgloss.Width(titleStyled)) / 2
	rightPad := borderWidth - leftPad - lipgloss.Width(titleStyled)

	topLine := fmt.Sprintf(
		"%s%s%s%s%s",
		topBorderLeft,
		strings.Repeat("─", leftPad),
		titleStyled,
		strings.Repeat("─", rightPad),
		topBorderRight,
	)

	// --- Build content area with vertical borders
	contentLines := strings.Split(m.viewport.View(), "\n")

	// The first wrapped line is the header (styled & padded safely)
	var wrapped []string
	wrapped = append(wrapped, fmt.Sprintf("│%s│", padLine(headerStyled, borderWidth)))

	for _, line := range contentLines {
		wrapped = append(wrapped, fmt.Sprintf("│%s│", padLine(line, borderWidth)))
	}

	// --- Bottom border
	bottomLine := fmt.Sprintf("╰%s╯", strings.Repeat("─", borderWidth))

	// --- Final render
	return strings.Join(append([]string{topLine}, append(wrapped, bottomLine)...), "\n")
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

// padLine ensures lines fit within the border width without slicing ANSI escapes.
// If the rendered width is <= width, pad with spaces. If it's larger, use lipgloss
// to safely truncate the styled string (preserving ANSI sequences).
func padLine(line string, width int) string {
	lineWidth := lipgloss.Width(line)
	if lineWidth == width {
		return line
	}
	if lineWidth < width {
		return line + strings.Repeat(" ", width-lineWidth)
	}
	// lineWidth > width: safely truncate styled string preserving escapes
	// Use MaxWidth to let lipgloss do the heavy lifting.
	return lipgloss.NewStyle().MaxWidth(width).Render(line)
}

// renderNodes builds the visible list of nodes with colorized header and cursor highlight.
func (m Model) renderNodes() string {
	if len(m.nodes) == 0 {
		return "No swarm nodes found."
	}

	var lines []string

	for i, n := range m.nodes {
		line := fmt.Sprintf("%-20s %-10s %-12s %-15s", n.Hostname, n.Status, n.Availability, n.ManagerStatus)
		if i == m.cursor {
			line = cursorStyle.Render(line)
		}
		lines = append(lines, line)
	}

	// Footer / status bar
	status := fmt.Sprintf(" Node %d of %d ", m.cursor+1, len(m.nodes))
	lines = append(lines, "", statusBarStyle.Render(status))

	return strings.Join(lines, "\n")
}
