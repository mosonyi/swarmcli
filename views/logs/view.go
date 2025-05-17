package logs

import (
	"fmt"
	"swarmcli/styles"
	"swarmcli/utils"
)

func (m Model) View() string {
	if !m.Visible {
		return ""
	}

	header := fmt.Sprintf("Inspecting (%s)", m.mode)
	if m.mode == "search" {
		header += fmt.Sprintf(" - Search: %s", m.searchTerm)
	}

	content := m.viewport.View()
	if len(m.searchMatches) > 0 {
		content = utils.HighlightMatches(content, m.searchTerm, m.searchMatches)
	}
	m.viewport.SetContent(content)

	return styles.BorderStyle.Render(
		fmt.Sprintf("%s\n\n%s\n\n[press q or esc to go back, / to search]", header, m.viewport.View()),
	)
}
