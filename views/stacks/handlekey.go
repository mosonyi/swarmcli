package stacksview

import (
	servicesview "swarmcli/views/services"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

// handleKey handles all key events for the stacks view.
func handleKey(m *Model, msg tea.KeyMsg) tea.Cmd {

	if m.mode == ModeSearching {
		switch msg.Type {
		case tea.KeyRunes:
			m.searchQuery += string(msg.Runes)
			m.applyFilter()
			m.viewport.SetContent(m.buildContent())
			return nil

		case tea.KeyBackspace:
			if len(m.searchQuery) > 0 {
				m.searchQuery = m.searchQuery[:len(m.searchQuery)-1]
			}
			m.applyFilter()
			m.viewport.SetContent(m.buildContent())
			return nil

		case tea.KeyEsc:
			m.mode = ModeNormal
			m.searchQuery = ""
			m.filtered = m.entries
			m.cursor = 0
			m.viewport.SetContent(m.buildContent())
			return nil
		}
	}

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
	case "/":
		m.mode = ModeSearching
		m.searchQuery = ""
		m.cursor = 0
		m.filtered = m.entries
		m.viewport.SetContent(m.buildContent())
		return nil
	}

	return nil
}
