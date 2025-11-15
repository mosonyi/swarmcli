package logsview

import (
	"fmt"
	"swarmcli/ui"
)

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}
	header := fmt.Sprintf("Inspecting Logs (%s)", m.mode)
	if m.mode == "search" {
		header += fmt.Sprintf(" - Search: %s", m.searchTerm)
	}
	footer := "[press q or esc to go back, / to search, f to toggle follow]"
	return ui.BorderStyle.Render(fmt.Sprintf("%s\n\n%s\n\n%s", header, m.viewport.View(), footer))
}
