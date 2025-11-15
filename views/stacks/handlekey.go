package stacksview

import (
	servicesview "swarmcli/views/services"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

// handleKey handles all key events for the stacks view.
func handleKey(m *Model, msg tea.KeyMsg) (*Model, tea.Cmd) {
	switch msg.String() {

	case "q", "esc":
		m.Visible = false
		return m, nil
	case "i", "enter":
		if m.cursor < len(m.entries) {
			selected := m.entries[m.cursor] // StackEntry
			return m, func() tea.Msg {
				return view.NavigateToMsg{
					ViewName: servicesview.ViewName,
					Payload: map[string]interface{}{
						"stackName": selected.Name,
					},
				}
			}
		}

	case "r":
		return m, LoadStacks(m.nodeID)

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.ensureCursorVisible()
			m.viewport.SetContent(m.buildContent())
		}
		return m, nil

	case "down", "j":
		if m.cursor < len(m.entries)-1 {
			m.cursor++
			m.ensureCursorVisible()
			m.viewport.SetContent(m.buildContent())
		}
		return m, nil

	case "pgup", "u":
		page := m.viewport.Height
		if m.cursor > page {
			m.cursor -= page
		} else {
			m.cursor = 0
		}
		m.ensureCursorVisible()
		m.viewport.SetContent(m.buildContent())
		return m, nil

	case "pgdown", "d":
		page := m.viewport.Height
		if m.cursor+page < len(m.entries) {
			m.cursor += page
		} else {
			m.cursor = len(m.entries) - 1
		}
		m.ensureCursorVisible()
		m.viewport.SetContent(m.buildContent())
		return m, nil
	}

	return m, nil
}
