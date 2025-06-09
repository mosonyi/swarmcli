package nodesview

import (
	"fmt"
	"swarmcli/styles"
)

func (m Model) View() string {
	if !m.Visible {
		return ""
	}

	header := "Nodes"

	return styles.BorderStyle.Render(
		fmt.Sprintf("%s\n\n%s\n\n[Press enter to see logs. Press q or esc to go back]", header, m.viewport.View()),
	)
}
