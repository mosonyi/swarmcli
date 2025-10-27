package nodesview

import (
	"fmt"
	"swarmcli/styles"
)

// View renders the nodes view.
func (m Model) View() string {
	if !m.Visible {
		return ""
	}

	header := "Nodes"

	content := fmt.Sprintf(
		"%s\n\n%s",
		header,
		m.viewport.View(),
	)

	return styles.BorderStyle.Render(content)
}
