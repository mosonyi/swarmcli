package systeminfoview

import "swarmcli/styles"

const Height = 6
const Width = 30

func (m Model) View() string {
	return styles.StatusStyle.Height(Height).
		Width(Width).
		Render(m.content)
}
