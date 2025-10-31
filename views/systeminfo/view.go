package systeminfoview

import "swarmcli/ui"

const Height = 6
const Width = 30

func (m Model) View() string {
	return ui.StatusStyle.Height(Height).
		Width(Width).
		Render(m.content)
}
