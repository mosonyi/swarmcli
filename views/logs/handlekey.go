package logsview

import (
	tea "github.com/charmbracelet/bubbletea"
)

// HandleKey handles keys for navigation and search (kept in same file for convenience)
func HandleKey(m Model, k tea.KeyMsg) (Model, tea.Cmd) {
	switch k.String() {
	case "q", "esc":
		// close view
		m.Visible = false
		// stop any streaming? we leave docker process to finish; we just drop UI
		return m, nil
	case "/":
		// enter search mode
		m.mode = "search"
		m.searchTerm = ""
		m.searchIndex = 0
		return m, nil
	case "enter":
		if m.mode == "search" {
			// commit search -> focus first match
			m.highlightContent()
			if len(m.searchMatches) > 0 {
				m.searchIndex = 0
				m.scrollToMatch()
			}
			m.mode = "normal"
			return m, nil
		}
	case "n":
		// next match
		if len(m.searchMatches) > 0 {
			m.searchIndex = (m.searchIndex + 1) % len(m.searchMatches)
			m.scrollToMatch()
		}
		return m, nil
	case "N":
		// prev match
		if len(m.searchMatches) > 0 {
			m.searchIndex = (m.searchIndex - 1 + len(m.searchMatches)) % len(m.searchMatches)
			m.scrollToMatch()
		}
		return m, nil
	case "up":
		m.viewport.LineUp(1)
		m.viewport.SetContent(m.buildContent())
		return m, nil
	case "down":
		m.viewport.LineDown(1)
		m.viewport.SetContent(m.buildContent())
		return m, nil
	case "pgup":
		m.viewport.LineUp(m.viewport.Height)
		m.viewport.SetContent(m.buildContent())
		return m, nil
	case "pgdown":
		m.viewport.LineDown(m.viewport.Height)
		m.viewport.SetContent(m.buildContent())
		return m, nil
	}

	// if in search mode, handle text input and backspace
	if m.mode == "search" {
		switch k.Type {
		case tea.KeyRunes:
			m.searchTerm += string(k.Runes)
			m.highlightContent()
			return m, nil
		case tea.KeyBackspace:
			if len(m.searchTerm) > 0 {
				m.searchTerm = m.searchTerm[:len(m.searchTerm)-1]
				m.highlightContent()
			}
			return m, nil
		}
	}

	return m, nil
}
