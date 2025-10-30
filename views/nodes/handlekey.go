package nodesview

import (
	"context"
	"fmt"
	"strings"
	"swarmcli/docker"
	inspectview "swarmcli/views/inspect"
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
			node := m.nodes[m.cursor]

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
	}

	return m, nil
}

func (m Model) renderNodes() string {
	var lines []string
	for _, n := range m.nodes {
		line := fmt.Sprintf("%-20s %-10s %-10s %-10s", n.Hostname, n.Status, n.Availability, n.ManagerStatus)
		lines = append(lines, line)
	}
	return strings.Join(lines, "\n")
}
