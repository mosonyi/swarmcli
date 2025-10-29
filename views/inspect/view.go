package inspectview

import (
	"fmt"
	"swarmcli/styles"
)

func (m Model) View() string {
	if !m.Visible {
		return ""
	}

	header := ""
	if m.title != "" {
		header = m.title
	} else {
		header = fmt.Sprintf("Inspecting (%s)", m.mode)
	}

	if m.mode == "search" {
		header += fmt.Sprintf(" - Search: %s", m.searchTerm)
	}

	return styles.BorderStyle.Render(
		fmt.Sprintf("%s\n\n%s[press q or esc to go back, / to search]", header, m.viewport.View()),
	)
}
