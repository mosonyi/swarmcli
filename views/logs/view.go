package logs

import "swarmcli/utils"

func (m Model) View() string {
	if !m.Visible {
		return ""
	}

	content := m.viewport.View()
	if len(m.searchMatches) > 0 {
		content = utils.HighlightMatches(content, m.searchTerm, m.searchMatches)
	}
	m.viewport.SetContent(content)

	footer := ""
	if m.mode == "search" {
		footer = "/" + m.searchTerm
	}

	return m.viewport.View() + "\n" + footer
}
