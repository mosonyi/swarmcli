package stacksview

import (
	servicesview "swarmcli/views/services"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

// handleKey handles all key events for the stacks view.
func handleKey(m *Model, msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {

	case "q", "esc":
		m.Visible = false
		return nil
	case "i", "enter":
		if m.cursor < len(m.entries) {
			selected := m.entries[m.cursor] // StackEntry
			return func() tea.Msg {
				return view.NavigateToMsg{
					ViewName: servicesview.ViewName,
					Payload: map[string]interface{}{
						"stackName": selected.Name,
					},
				}
			}
		}

	case "r":
		return LoadStacks(m.nodeID)

	case "up", "k":
		if m.cursor > 0 {
			m.cursor--
			m.ensureCursorVisible()
			m.viewport.SetContent(m.buildContent())
		}
		return nil

	case "down", "j":
		if m.cursor < len(m.entries)-1 {
			m.cursor++
			m.ensureCursorVisible()
			m.viewport.SetContent(m.buildContent())
		}
		return nil

	case "pgup", "u":
		page := m.viewport.Height
		if m.cursor > page {
			m.cursor -= page
		} else {
			m.cursor = 0
		}
		m.ensureCursorVisible()
		m.viewport.SetContent(m.buildContent())
		return nil

	case "pgdown", "d":
		page := m.viewport.Height
		if m.cursor+page < len(m.entries) {
			m.cursor += page
		} else {
			m.cursor = len(m.entries) - 1
		}
		m.ensureCursorVisible()
		m.viewport.SetContent(m.buildContent())
		return nil
	}

	return nil
}
