package nodesview

import (
	"strings"
	inspectview "swarmcli/views/inspect"
	stacksview "swarmcli/views/stacks"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func HandleKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	return handleNormalModeKey(m, msg)
}

func handleNormalModeKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		m.Visible = false

	case "j", "down":
		if m.cursor < len(m.nodes)-1 {
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

	case "i":
		if m.cursor < len(m.nodes) {
			return m, func() tea.Msg {
				return view.NavigateToMsg{
					ViewName: inspectview.ViewName,
					Payload:  m.nodes[m.cursor],
				}
			}
		}

	case "s":
		return m.selectNode()
	}

	return m, nil
}

func (m Model) selectNode() (Model, tea.Cmd) {
	fields := strings.Fields(m.nodes[m.cursor])
	if len(fields) == 0 {
		return m, nil
	}

	nodeID := fields[0]
	return m, func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: stacksview.ViewName,
			Payload:  nodeID,
		}
	}
}
