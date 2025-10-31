package stacksview

import (
	logsview "swarmcli/views/logs"
	"swarmcli/views/view"

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
		if m.cursor < len(m.entries)-1 {
			m.cursor++
			m.viewport.SetContent(m.buildContent())
		}
	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.viewport.SetContent(m.buildContent())
		}
	case "pgup":
		m.viewport.ScrollUp(m.viewport.Height)
	case "pgdown":
		m.viewport.ScrollDown(m.viewport.Height)
	case "enter":
		if m.cursor < len(m.entries) {
			serviceID := m.entries[m.cursor]
			return m, func() tea.Msg {
				return view.NavigateToMsg{
					ViewName: logsview.ViewName,
					Payload:  serviceID.Name,
				}
			}
		}
	}
	return m, nil
}
