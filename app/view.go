package app

import (
	"swarmcli/views/helpbar"
	systeminfoview "swarmcli/views/systeminfo"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	// Check if current view has fullscreen mode enabled
	if logsView, ok := m.currentView.(interface{ GetFullscreen() bool }); ok && logsView.GetFullscreen() {
		// Fullscreen mode: show only the current view (no helpbar, no stackbar)
		return m.currentView.View()
	}

	systemInfo := m.systemInfo.View()

	help := helpbar.New(m.viewport.Width, systeminfoview.Height).
		WithGlobalHelp([]helpbar.HelpEntry{{Key: "q", Desc: "quit"}, {Key: "?", Desc: "help"}}).
		WithViewHelp(m.currentView.ShortHelpItems()).
		View(systemInfo)

	if m.commandInput.Visible() {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			help,
			m.commandInput.View(),
			m.currentView.View(),
			m.renderStackBar(),
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		help,
		m.currentView.View(),
		m.renderStackBar(),
	)
}
