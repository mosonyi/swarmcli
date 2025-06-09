package nodesview

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	stacksview "swarmcli/views/stacks"
)

func HandleKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	return handleModeKey(m, msg)
}

func handleModeKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "j", "down":
		if m.cursor < len(m.nodes)-1 {
			m.cursor++
		}

	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}

	case "i":
		if m.cursor < len(m.nodes) {
			cmd := inspectItem(m.nodes[m.cursor])
			return m, cmd
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
	return m, stacksview.LoadNodeStacks(nodeID)
}
