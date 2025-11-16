package inspectview

import (
	"fmt"
	"swarmcli/ui"
)

func (m *Model) View() string {
	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}

	// ---- Build dynamic title ----
	title := m.Title
	if title == "" {
		title = "Inspecting"
	}

	// ---- Build header ----
	header := "Inspecting"
	if m.searchMode {
		header = fmt.Sprintf("%s — Search: %s", header, m.SearchTerm)
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
