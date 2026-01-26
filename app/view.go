// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package app

import (
	"swarmcli/ui"
	"swarmcli/views/helpbar"
	systeminfoview "swarmcli/views/systeminfo"
	"swarmcli/views/view"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	// Check if current view has fullscreen mode enabled
	if logsView, ok := m.currentView.(interface{ GetFullscreen() bool }); ok && logsView.GetFullscreen() {
		// Fullscreen mode: show only the current view (no helpbar, no stackbar)
		return m.currentView.View()
	}

	systemInfo := m.systemInfo.View()

	// Build global help - exclude "?" when already in help view
	globalHelp := []helpbar.HelpEntry{{Key: "?", Desc: "Help"}}
	if m.currentView.Name() == view.NameHelp {
		globalHelp = []helpbar.HelpEntry{}
	}

	help := helpbar.New(m.viewport.Width, systeminfoview.Height).
		WithGlobalHelp(globalHelp).
		WithViewHelp(m.currentView.ShortHelpItems()).
		View(systemInfo)

	if m.commandInput.Visible() {
		// Render a framed 3-line command box between the header and main view.
		// Use the viewport width (which is usable width) and add 4 to match
		// the frame sizing used by views that render full-width frames.
		frameWidth := m.viewport.Width + 4
		// Render the normal framed command box then post-process the top
		// border to replace corner glyphs so it visually integrates with
		// the header above.
		cmdFrame := ui.RenderFramedBoxHeight("", "", m.commandInput.View(), "", frameWidth, 3)

		// Use the framed command box as rendered (keep corner glyphs)

		return lipgloss.JoinVertical(
			lipgloss.Left,
			help,
			cmdFrame,
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
