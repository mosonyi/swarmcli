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
		Foreground(lipgloss.Color("81")).
		Bold(true).
		Underline(true)

	cursorStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("230")).
		Background(lipgloss.Color("63")).
		Bold(true)

	statusBarStyle = lipgloss.NewStyle().
		Foreground(lipgloss.Color("250")).
		Background(lipgloss.Color("237")).
		Padding(0, 1)
)

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

	title := fmt.Sprintf(" Nodes (%d total, %d manager%s) ", total, managers, plural(managers))
	titleRendered := titleStyle.Render(title)

	// Define subtle style for the column legend
	subtleStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")) // grayish for secondary info

	// Column header like in k9s
	header := subtleStyle.Render("HOSTNAME              STATUS     AVAILABILITY   MANAGER STATUS")

	// Main table body from the viewport
	body := m.viewport.View()

	// Compute total content width
	contentWidth := lipgloss.Width(header)
	if w := lipgloss.Width(body); w > contentWidth {
		contentWidth = w
	}

	// Build the top border with title embedded inside
	topBorder := fmt.Sprintf("╭%s╮", padTitleInBorder(titleRendered, contentWidth))

	// Build body lines wrapped in vertical borders
	bodyLines := strings.Split(fmt.Sprintf("%s\n%s", header, body), "\n")
	for i, line := range bodyLines {
		bodyLines[i] = fmt.Sprintf("│%-*s│", contentWidth, line)
	}

	// Bottom border
	bottomBorder := "╰" + strings.Repeat("─", contentWidth) + "╯"

	// Join everything together
	return strings.Join(append([]string{topBorder}, append(bodyLines, bottomBorder)...), "\n")
}

// padTitleInBorder places the title neatly inside the top border, like k9s does.
func padTitleInBorder(title string, totalWidth int) string {
	titleWidth := lipgloss.Width(title)
	if titleWidth >= totalWidth {
		return title
	}
	paddingRight := totalWidth - titleWidth
	return title + strings.Repeat("─", paddingRight)
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
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
