package inspectview

import (
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func handleNormalKey(m Model, k tea.KeyMsg) (view.View, tea.Cmd) {
	switch k.String() {
	case "q", "esc":
		//m.Visible = false
		return m, nil
	case "/", "shift+/":
		m.searchMode = true
		m.SearchTerm = ""
		return m, nil
	case "up", "k":
		if m.Cursor > 0 {
			m.Cursor--
		}
		m.scrollToCursor()
		return m, nil
	case "down", "j":
		if m.Cursor < len(m.Visible)-1 {
			m.Cursor++
		}
		m.scrollToCursor()
		return m, nil
	case "left", "h":
		// collapse current node or move to parent
		if len(m.Visible) == 0 {
			return m, nil
		}
		node := m.Visible[m.Cursor]
		if node.Expanded && len(node.Children) > 0 {
			node.Expanded = false
		} else if node.Parent != nil {
			// move cursor to parent
			par := node.Parent
			// find parent index in visible
			for i, nn := range m.Visible {
				if nn == par {
					m.Cursor = i
					break
				}
			}
		}
		m.rebuildVisible()
		m.scrollToCursor()
		return m, nil
	case "right", "l", "enter":
		if len(m.Visible) == 0 {
			return m, nil
		}
		node := m.Visible[m.Cursor]
		if len(node.Children) > 0 {
			node.Expanded = true
			m.rebuildVisible()
			m.scrollToCursor()
		}
		return m, nil
	case " ", "":
		// toggle expand/collapse
		if len(m.Visible) == 0 {
			return m, nil
		}
		node := m.Visible[m.Cursor]
		if len(node.Children) > 0 {
			node.Expanded = !node.Expanded
			m.rebuildVisible()
			m.scrollToCursor()
		}
		return m, nil
	case "n":
		// next match
		if m.SearchTerm == "" || len(m.Visible) == 0 {
			return m, nil
		}
		start := m.Cursor + 1
		for i := start; i < len(m.Visible); i++ {
			if m.Visible[i].Matches {
				m.Cursor = i
				m.scrollToCursor()
				return m, nil
			}
		}
		// wrap
		for i := 0; i <= m.Cursor; i++ {
			if m.Visible[i].Matches {
				m.Cursor = i
				m.scrollToCursor()
				return m, nil
			}
		}
		return m, nil
	case "N":
		// prev match
		if m.SearchTerm == "" || len(m.Visible) == 0 {
			return m, nil
		}
		for i := m.Cursor - 1; i >= 0; i-- {
			if m.Visible[i].Matches {
				m.Cursor = i
				m.scrollToCursor()
				return m, nil
			}
		}
		// wrap
		for i := len(m.Visible) - 1; i > m.Cursor; i-- {
			if m.Visible[i].Matches {
				m.Cursor = i
				m.scrollToCursor()
				return m, nil
			}
		}
		return m, nil
	}
	return m, nil
}

func handleSearchKey(m Model, k tea.KeyMsg) (view.View, tea.Cmd) {
	switch k.Type {
	case tea.KeyRunes:
		m.SearchTerm += k.String()
		// live preview of matches
		m.rebuildVisible()
		m.viewport.SetContent(m.renderVisible())
	case tea.KeyBackspace:
		if len(m.SearchTerm) > 0 {
			m.SearchTerm = m.SearchTerm[:len(m.SearchTerm)-1]
			m.rebuildVisible()
			m.viewport.SetContent(m.renderVisible())
		}
	case tea.KeyEnter:
		// finalize search: cursor already moved to first match by rebuildVisible
		m.searchMode = false
		m.viewport.SetContent(m.renderVisible())
		m.scrollToCursor()
	case tea.KeyEsc:
		// cancel search
		m.searchMode = false
		m.SearchTerm = ""
		// collapse all nodes? we keep expanded state but rebuild normal view
		m.rebuildVisible()
		m.viewport.SetContent(m.renderVisible())
	}
	return m, nil
}

// scrollToCursor ensures the cursor is visible (centers it when possible)
func (m *Model) scrollToCursor() {
	if m.Cursor < 0 || m.Cursor >= len(m.Visible) {
		return
	}
	line := m.Cursor
	offset := line - m.viewport.Height/2
	if offset < 0 {
		offset = 0
	}
	m.viewport.GotoTop()
	m.viewport.SetYOffset(offset)
	m.viewport.SetContent(m.renderVisible())
}
