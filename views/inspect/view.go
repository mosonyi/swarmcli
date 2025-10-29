package inspectview

import (
	"fmt"
	"swarmcli/styles"
)

func (m Model) View() string {
	if m.Root == nil && !m.ready {
		return ""
	}
	if m.Title == "" {
		if m.searchMode {
			m.Title = fmt.Sprintf("Inspecting (search: %s)", m.SearchTerm)
		} else {
			m.Title = "Inspecting"
		}
	}
	header := m.Title
	if m.searchMode {
		header = fmt.Sprintf("%s  (type to search, enter to apply, esc to cancel)", header)
	}
	out := fmt.Sprintf("%s\n\n%s[press q or esc to go back, / to search]", header, m.viewport.View())
	return styles.BorderStyle.Render(out)
}
