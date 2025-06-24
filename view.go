package main

import (
	"github.com/charmbracelet/lipgloss"
	"swarmcli/views/helpbar"
	systeminfoview "swarmcli/views/systeminfo"
)

func (m model) View() string {
	systemInfo := m.systemInfo.View()

	help := helpbar.New(m.viewport.Width, systeminfoview.Height).
		WithGlobalHelp([]helpbar.HelpEntry{{Key: "q", Desc: "quit"}, {Key: "?", Desc: "help"}}).
		WithViewHelp(m.currentView.ShortHelpItems()).
		View(systemInfo)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		help,
		m.currentView.View(),
		m.renderStackBar(),
	)
}
