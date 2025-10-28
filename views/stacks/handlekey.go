package stacksview

import (
	"swarmcli/docker"
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
	case "r":
		return m, refreshStacksCmd(m.nodeId)
	case "q", "esc":
		m.Visible = false
	case "j", "down":
		if m.stackCursor < len(m.stacks)-1 {
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
		if m.stackCursor < len(m.stacks) {
			serviceID := m.stacks[m.stackCursor]
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

func refreshStacksCmd(nodeID string) tea.Cmd {
	return func() tea.Msg {
		// Refresh hostname cache first
		if err := docker.RefreshHostnameCache(); err != nil {
			return RefreshErrorMsg{Err: err}
		}

		// Fetch stacks for node or all nodes
		stacks := docker.GetStacks(nodeID)
		return Msg{
			NodeId: nodeID,
			Stacks: stacks,
		}
	}
}
