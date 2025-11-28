package logsview

import (
	"fmt"
	"swarmcli/ui"

	"github.com/charmbracelet/lipgloss"
)

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}

	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}

	// ---- Build dynamic title bar ----
	followStatus := "off"
	if m.getFollow() {
		followStatus = "on"
	}
	wrapStatus := "off"
	if m.getWrap() {
		wrapStatus = "on"
	}

	title := fmt.Sprintf(
		"Service: %s • AutoScroll: %s • wrap: %s",
		m.ServiceEntry.ServiceName,
		followStatus,
		wrapStatus,
	)

	// ---- Build header ----
	header := "Logs"
	if m.mode == "search" {
		header = fmt.Sprintf("Logs — Search: %s", m.searchTerm)
	}

	headerRendered := ui.FrameHeaderStyle.Render(header)

	// ---- Render based on fullscreen mode ----
	if m.fullscreen {
		// In fullscreen: show centered title at top with same styling as frame title
		titleText := ui.FrameTitleStyle.Render(title)
		titleStyle := lipgloss.NewStyle().
			Width(width).
			Align(lipgloss.Center)
		titleRendered := titleStyle.Render(titleText)
		
		// If in search mode, show search header on second line
		if m.mode == "search" {
			searchHeader := ui.FrameHeaderStyle.Render(header)
			return titleRendered + "\n" + searchHeader + "\n" + m.viewport.View()
		}
		
		return titleRendered + "\n" + m.viewport.View()
	}

	// ---- Normal mode: render framed box ----
	content := ui.RenderFramedBox(
		title,
		headerRendered,
		m.viewport.View(),
		"",
		width,
	)

	return content
}
