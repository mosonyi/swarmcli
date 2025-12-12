package app

import (
	"fmt"
	"strings"
	"swarmcli/ui"
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

	// Build a fixed-height header block consisting of systemInfo + helpbar,
	// ensuring it occupies exactly systeminfoview.Height lines.
	systemInfo := m.systemInfo.View()
	helpBar := helpbar.New(m.viewport.Width, systeminfoview.Height).
		WithGlobalHelp([]helpbar.HelpEntry{{Key: "?", Desc: "Help"}}).
		WithViewHelp(m.currentView.ShortHelpItems()).
		View(systemInfo)

	// helpBar already includes systemInfo. Clamp/pad helpBar to the fixed
	// header height so the top area always occupies the expected lines.
	hl := strings.Split(helpBar, "\n")
	if len(hl) > systeminfoview.Height {
		hl = hl[:systeminfoview.Height]
	} else if len(hl) < systeminfoview.Height {
		for i := 0; i < systeminfoview.Height-len(hl); i++ {
			hl = append(hl, "")
		}
	}
	help := strings.Join(hl, "\n")

	if m.commandInput.Visible() {
		return lipgloss.JoinVertical(
			lipgloss.Left,
			help,
			m.commandInput.View(),
			m.currentView.View(),
			m.renderStackBar(),
		)
	}

	// Add an autorunning bottom line showing sizes: terminal, usable viewport, expected view height
	// Compute expected view height the same way as handleViewResize
	isFullscreen := false
	if logsView, ok := m.currentView.(interface{ GetFullscreen() bool }); ok {
		isFullscreen = logsView.GetFullscreen()
	}
	var expectedViewHeight int
	if isFullscreen {
		expectedViewHeight = m.viewport.Height - 1
	} else {
		expectedViewHeight = m.viewport.Height - systeminfoview.Height
	}
	bottomLine := ui.StatusBarStyle.Render(fmt.Sprintf("Max:%d usable:%d viewH:%d", m.terminalHeight, m.viewport.Height, expectedViewHeight))

	return lipgloss.JoinVertical(
		lipgloss.Left,
		help,
		m.currentView.View(),
		m.renderStackBar(),
		bottomLine,
	)
}
