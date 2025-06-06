package stacksview

import (
	tea "github.com/charmbracelet/bubbletea"
)

func HandleKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	// Maybe add search mode handling in the future
	return handleNormalModeKey(m, msg)
}

func handleNormalModeKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "q", "esc":
		m.Visible = false
	case "j", "down":
		if m.stackCursor < len(m.nodeStacks)-1 {
			m.stackCursor++
		}
	case "k", "up":
		if m.stackCursor > 0 {
			m.stackCursor--
		}
	case "pgup":
		m.viewport.ScrollUp(m.viewport.Height)
	case "pgdown":
		m.viewport.ScrollDown(m.viewport.Height)
	}
	return m, nil
}
