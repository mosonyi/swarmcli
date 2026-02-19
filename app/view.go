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
	// Fullscreen mode: show only the current view (no helpbar, no stackbar)
	if m.fullscreen {
		title := m.currentView.FrameTitle()
		header := m.currentView.FrameHeader()
		content := m.currentView.FrameContent()
		return ui.RenderViewFrame(title, header, content, "", m.terminalWidth, m.terminalHeight, true)
	}

	systemInfo := m.systemInfo.View()

	// Build global help - exclude "?" when already in help view
	globalHelp := []helpbar.HelpEntry{
		{Key: "f", Desc: "Fullscreen"},
		{Key: "?", Desc: "Help"},
	}
	if m.currentView.Name() == view.NameHelp {
		globalHelp = []helpbar.HelpEntry{}
	}

	// Check if current view has errors for logo color
	hasError := m.currentView.HasErrors()

	help := helpbar.New(m.viewport.Width, systeminfoview.Height).
		WithGlobalHelp(globalHelp).
		WithViewHelp(m.currentView.ShortHelpItems()).
		View(systemInfo, hasError)

	// Build the framed view from components
	title := m.currentView.FrameTitle()
	header := m.currentView.FrameHeader()
	content := m.currentView.FrameContent()
	content = ui.LeftPadContent(content)
	footer := m.currentView.FrameFooter()
	// Subtract help bar (systeminfoview.Height) and stack bar (1 line) from the
	// viewport height so the frame fits between chrome elements.
	frameHeight := m.viewport.Height - systeminfoview.Height - 1
	if frameHeight < 1 {
		frameHeight = 1
	}
	framedView := ui.RenderViewFrame(title, header, content, footer, m.viewport.Width, frameHeight, false)

	if m.commandInput.Visible() {
		// Render a framed 3-line command box between the header and main view.
		frameWidth := m.viewport.Width + 4
		cmdFrame := ui.RenderFramedBoxHeight("", "", m.commandInput.View(), "", frameWidth, 3)

		return lipgloss.JoinVertical(
			lipgloss.Left,
			help,
			cmdFrame,
			framedView,
			m.renderStackBar(),
		)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		help,
		framedView,
		m.renderStackBar(),
	)
}
