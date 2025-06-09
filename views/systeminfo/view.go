package systeminfoview

import "swarmcli/styles"

func (m Model) View() string {
	return styles.StatusStyle.Height(6).
		Render(m.content)
}
