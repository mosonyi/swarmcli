package inspectview

import (
	tea "github.com/charmbracelet/bubbletea"
)

func HandleKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	if m.mode == "search" {
		return handleSearchModeKey(m, msg)
	}
	return handleNormalModeKey(m, msg)
}

func handleSearchModeKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyRunes:
		m.searchTerm += msg.String()
		m.highlightContent()
	case tea.KeyBackspace:
		if len(m.searchTerm) > 0 {
			m.searchTerm = m.searchTerm[:len(m.searchTerm)-1]
			m.highlightContent()
		}
	case tea.KeyEnter:
		m.mode = "normal"
		m.highlightContent()
		m.searchIndex = 0
		m.scrollToMatch()
	case tea.KeyEsc:
		m.mode = "normal"
	}
	return m, nil
}

func handleNormalModeKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.Visible = false
	case "/":
		m.mode = "search"
		m.searchTerm = ""
		// Restore unhighlighted content
		m.viewport.SetContent(m.inspectLines)
		m.searchMatches = nil
		m.searchIndex = 0
	case "n":
		if len(m.searchMatches) > 0 {
			m.searchIndex = (m.searchIndex + 1) % len(m.searchMatches)
			m.scrollToMatch()
		}
	case "N":
		if len(m.searchMatches) > 0 {
			m.searchIndex = (m.searchIndex - 1 + len(m.searchMatches)) % len(m.searchMatches)
			m.scrollToMatch()
		}
	case "up":
		m.viewport.ScrollUp(1)
	case "down":
		m.viewport.ScrollDown(1)
	case "pgup":
		m.viewport.ScrollUp(m.viewport.Height)
	case "pgdown":
		m.viewport.ScrollDown(m.viewport.Height)
	}
	return m, nil
}
