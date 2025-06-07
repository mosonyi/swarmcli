package stacksview

import (
	tea "github.com/charmbracelet/bubbletea"
	"swarmcli/views/logs"
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
		if m.stackCursor < len(m.stackServices)-1 {
			m.stackCursor++
			m.viewport.SetContent(m.buildContent())
		}
	case "k", "up":
		if m.stackCursor > 0 {
			m.stackCursor--
			m.viewport.SetContent(m.buildContent())
		}
	case "pgup":
		m.viewport.ScrollUp(m.viewport.Height)
	case "pgdown":
		m.viewport.ScrollDown(m.viewport.Height)
	case "enter":
		if m.stackCursor < len(m.stackServices) {
			serviceID := m.stackServices[m.stackCursor]
			return m, logs.Load(serviceID.ServiceName)
		}
	}
	return m, nil
}
