package stacksview

import (
	"fmt"
	"swarmcli/styles"
)

func (m Model) View() string {
	if !m.Visible {
		return ""
	}

	header := fmt.Sprintf("Stacks on Node")

	return styles.BorderStyle.Render(
		fmt.Sprintf("%s\n\n%s\n\n[press q or esc to go back]", header, m.viewport.View()),
	)
}
