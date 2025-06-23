package systeminfoview

import "swarmcli/styles"

const Height = 6

func (m Model) View() string {
	return styles.StatusStyle.Height(Height).
		Render(m.content)
}
