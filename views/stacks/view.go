package stacksview

import (
	"fmt"
	"strings"
	"swarmcli/ui"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}

	total := len(m.entries)
	title := fmt.Sprintf("Stacks on Node (Total: %d)", total)

	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}

	// --- Header Style ---
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("12")) // light blue

	stackColWidth := m.computeStackColWidth(width)
	serviceColWidth := 10 // “Services” column is narrow and fixed
	header := headerStyle.Render(fmt.Sprintf(
		"%-*s %*s",
		stackColWidth, "STACK",
		serviceColWidth, "SERVICES",
	))

	content := m.viewport.View()

	return ui.RenderFramedBox(title, header, content, width)
}

// --- Internal Rendering ---

func (m *Model) buildContent() string {
	if len(m.entries) == 0 {
		return "No stacks found for this node."
	}

	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}

	stackColWidth := m.computeStackColWidth(width)
	serviceColWidth := 10

	var lines []string
	for i, s := range m.entries {
		line := fmt.Sprintf("%-*s %*d", stackColWidth, s.Name, serviceColWidth, s.ServiceCount)
		if i == m.cursor {
			line = ui.CursorStyle.Render(line)
		}
		lines = append(lines, line)
	}

	status := fmt.Sprintf(" Stack %d of %d ", m.cursor+1, len(m.entries))
	lines = append(lines, "", ui.StatusBarStyle.Render(status))

	return strings.Join(lines, "\n")
}

// computeStackColWidth dynamically adjusts the column width based on viewport width and data.
func (m *Model) computeStackColWidth(totalWidth int) int {
	const minWidth = 15
	const gap = 2
	serviceCol := 10

	available := totalWidth - serviceCol - gap
	if available < minWidth {
		return minWidth
	}

	maxName := minWidth
	for _, s := range m.entries {
		if l := len(s.Name); l > maxName {
			maxName = l
		}
	}

	if maxName+gap < available {
		return maxName + gap
	}

	return available
}
