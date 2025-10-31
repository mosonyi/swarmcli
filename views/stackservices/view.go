package stackservicesview

import (
	"fmt"
	"strings"
	"swarmcli/ui"

	"github.com/charmbracelet/lipgloss"
)

var (
	cursorStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("230")).
			Background(lipgloss.Color("63")).
			Bold(true)
	headerStyle = lipgloss.NewStyle().Bold(true)
)

func (m Model) View() string {
	if !m.Visible {
		return ""
	}
	header := "STACK NAME           SERVICE NAME"
	content := m.viewport.View()
	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}
	return ui.RenderFramedBox(m.title, header, content, width)
}

func (m Model) renderEntries() string {
	if len(m.entries) == 0 {
		return "No services found."
	}

	var lines []string
	for i, e := range m.entries {
		line := fmt.Sprintf("%-20s %-20s", e.StackName, e.ServiceName)
		if i == m.cursor {
			line = cursorStyle.Render(line)
		}
		lines = append(lines, line)
	}
	status := fmt.Sprintf(" Service %d of %d ", m.cursor+1, len(m.entries))
	lines = append(lines, "", headerStyle.Render(status))
	return strings.Join(lines, "\n")
}
