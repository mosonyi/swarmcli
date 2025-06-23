package stacksview

import (
	"fmt"
	"swarmcli/styles"
)

func (m Model) View() string {
	if !m.Visible {
		return ""
	}

	header := fmt.Sprintf("Stacks on Node: %s", m.nodeId)

	return styles.BorderStyle.Render(
		fmt.Sprintf("%s\n\n%s[Press enter to see logs. Press q or esc to go back]", header, m.viewport.View()),
	)
}
