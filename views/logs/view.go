package logsview

import (
	"fmt"
	"swarmcli/ui"
)

func (m Model) View() string {
	if !m.Visible {
		return ""
	}

	header := fmt.Sprintf("Inspecting Logs (%s)", m.mode)
	if m.mode == "search" {
		header += fmt.Sprintf(" - Search: %s", m.searchTerm)
	}

	return ui.BorderStyle.Render(
		fmt.Sprintf("%s\n\n%s[press q or esc to go back, / to search]", header, m.viewport.View()),
	)
}
