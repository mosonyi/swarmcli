package inspectview

import (
	"fmt"
	"swarmcli/ui"
)

func (m *Model) View() string {
	if m.Root == nil && !m.ready {
		return ""
	}

	// build title
	title := m.Title
	if title == "" {
		title = "Inspecting"
	}

	// if in search mode, show the search input
	if m.searchMode {
		title = fmt.Sprintf("%s  (search: %s, enter to apply, esc to cancel)", title, m.SearchTerm)
	}

	content := m.viewport.View()
	out := fmt.Sprintf("%s\n\n%s\n[press q or esc to go back, / to search]", title, content)
	return ui.BorderStyle.Render(out)
}
