package nodesview

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	stacksview "swarmcli/views/stacks"
)

func HandleKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
	// Maybe add search mode handling in the future
	return handleNormalModeKey(m, msg)
}

func handleNormalModeKey(m Model, msg tea.KeyMsg) (Model, tea.Cmd) {
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

	//case ":":
	//	m.commandMode = true

	case "s":
		return m.handleSelectNode()

	}
	return m, nil
}

func (m Model) handleSelectNode() (Model, tea.Cmd) {
	fields := strings.Fields(m.nodes[m.cursor])
	if len(fields) == 0 {
		return m, nil
	}

	nodeID := fields[0]
	return m, stacksview.LoadNodeStacks(nodeID)
}
