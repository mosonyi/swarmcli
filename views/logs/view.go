package logsview

import (
	"fmt"
	"swarmcli/ui"
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
	if m.follow {
		followStatus = "on"
	}

	title := fmt.Sprintf(
		"Logs • %s • follow: %s",
		m.ServiceEntry.ServiceName,
		followStatus,
	)

	// ---- Build header ----
	header := "Logs"
	if m.mode == "search" {
		header = fmt.Sprintf("Logs — Search: %s", m.searchTerm)
	}

	headerRendered := ui.FrameHeaderStyle.Render(header)

	// ---- Render framed box ----
	content := ui.RenderFramedBox(
		title,
		headerRendered,
		m.viewport.View(),
		width,
	)

	return content
}
