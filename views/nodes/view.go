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
		fmt.Sprintf("%s\n\n%s", header, m.viewport.View()),
	)
}
