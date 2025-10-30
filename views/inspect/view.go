package inspectview

import (
	"fmt"
	"swarmcli/styles"
)

func (m Model) View() string {
	if m.Root == nil && !m.ready {
		return ""
	}

	title := m.Title
	if title == "" {
		if m.searchMode {
			title = fmt.Sprintf("Inspecting (search: %s)", m.SearchTerm)
		} else {
			title = "Inspecting"
		}
	}

	content := m.viewport.View()
	out := fmt.Sprintf("%s\n\n%s\n[press q or esc to go back, / to search]", title, content)
	return styles.BorderStyle.Render(out)
}
