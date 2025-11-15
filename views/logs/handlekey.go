package logsview

import (
	tea "github.com/charmbracelet/bubbletea"
)

func HandleKey(m *Model, k tea.KeyMsg) tea.Cmd {
	switch k.String() {
	case "q", "esc":
		m.Visible = false
		return nil
	case "/":
		m.mode = "search"
		m.searchTerm = ""
		m.searchIndex = 0
		return nil
	case "enter":
		if m.mode == "search" {
			m.highlightContent()
			if len(m.searchMatches) > 0 {
				m.searchIndex = 0
				m.scrollToMatch()
			}
			m.mode = "normal"
			return nil
		}
	case "n":
		if len(m.searchMatches) > 0 {
			m.searchIndex = (m.searchIndex + 1) % len(m.searchMatches)
			m.scrollToMatch()
		}
		return nil
	case "N":
		if len(m.searchMatches) > 0 {
			m.searchIndex = (m.searchIndex - 1 + len(m.searchMatches)) % len(m.searchMatches)
			m.scrollToMatch()
		}
		return nil
	case "f":
		// toggle follow mode
		m.setFollow(!m.follow)
		l().Debugf("[logsview] follow toggled -> %v", m.follow)
		return nil
	}

	// if in search mode, capture runes/backspace
	if m.mode == "search" {
		switch k.Type {
		case tea.KeyRunes:
			m.searchTerm += string(k.Runes)
			m.highlightContent()
			return nil
		case tea.KeyBackspace:
			if len(m.searchTerm) > 0 {
				m.searchTerm = m.searchTerm[:len(m.searchTerm)-1]
				m.highlightContent()
			}
			return nil
		}
	}

	return nil
}
