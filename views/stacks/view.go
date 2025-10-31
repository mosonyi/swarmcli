package stacksview

import (
	"fmt"
	"swarmcli/ui"
)

func (m Model) View() string {
	if !m.Visible {
		return ""
	}

	header := "Stacks"
	if m.nodeId != "" {
		header = fmt.Sprintf("Stacks on Node: %s", m.nodeId)
	}

	return ui.BorderStyle.Render(
		fmt.Sprintf("%s\n\n%s[Press enter to see logs. Press q or esc to go back]",
			header, m.viewport.View()),
	)
}
