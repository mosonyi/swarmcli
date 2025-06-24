package main

import (
	"github.com/charmbracelet/lipgloss"
	"swarmcli/views/helpbar"
	systeminfoview "swarmcli/views/systeminfo"
)

func (m model) View() string {
	//helpText := styles.HelpStyle.Render("[i: inspect, s: see stacks, q: quit, j/k: move cursor, : switch mode]")

	systemInfo := m.systemInfo.View()

	help := helpbar.New(m.viewport.Width, systeminfoview.Height).
		WithGlobalHelp([]helpbar.HelpEntry{{"q", "quit"}, {"?", "help"}}).
		WithViewHelp(m.currentView.ShortHelpItems()).
		View(systemInfo)

	return lipgloss.JoinVertical(
		lipgloss.Left,
		help,
		m.currentView.View(),
		m.renderStackBar(),
	)
}
