package nodesview

import (
	"context"
	"fmt"
	"swarmcli/docker"
	inspectview "swarmcli/views/inspect"
	nodeservicesview "swarmcli/views/nodeservices"
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
		if m.cursor < len(m.entries)-1 {
			m.cursor++
			m.viewport.SetContent(m.renderNodes())
		}

	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
			m.viewport.SetContent(m.renderNodes())
		}

	case "pgup":
		m.viewport.ScrollUp(m.viewport.Height)

	case "pgdown":
		m.viewport.ScrollDown(m.viewport.Height)

	case "d":
		if m.cursor < len(m.entries) {
			node := m.entries[m.cursor]

			return m, func() tea.Msg {
				inspectContent, err := docker.Inspect(context.Background(), docker.InspectNode, node.ID)
				if err != nil {
					inspectContent = fmt.Sprintf("Error inspecting node %q: %v", node.ID, err)
				}

				return view.NavigateToMsg{
					ViewName: inspectview.ViewName,
					Payload: map[string]interface{}{
						"title": fmt.Sprintf("Node: %s", node.Hostname),
						"json":  inspectContent,
					},
				}
			}
		}
	case "i":
		if m.cursor < len(m.entries) {
			node := m.entries[m.cursor]
			return m, func() tea.Msg {
				return view.NavigateToMsg{
					ViewName: nodeservicesview.ViewName,
					Payload: map[string]interface{}{
						"nodeID":   node.ID,
						"hostname": node.Hostname,
					},
				}
			}
		}
	}

	return m, nil
}
