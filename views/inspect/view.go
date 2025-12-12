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

	// ---- Format indicator ----
	formatIndicator := "[YAML]"
	if m.Format == "raw" {
		formatIndicator = "[RAW]"
	}

	// ---- Build header ----
	errorHint := ""
	if m.ParseError != "" {
		errorHint = " — Could not parse JSON, showing raw"
	}

	header := fmt.Sprintf("Inspecting %s%s", formatIndicator, errorHint)

	if m.searchMode {
		header = fmt.Sprintf("%s — Search: %s", header, m.SearchTerm)
	}

	headerRendered := ui.FrameHeaderStyle.Render(header)

	// Add 4 to make frame full terminal width (app reduces viewport by 4 in normal mode)
	frameWidth := width + 4
	frameHeight := m.viewport.Height
	if frameHeight < 0 {
		frameHeight = 0
	}

	// ---- Render framed box ----
	content := ui.RenderFramedBoxHeight(
		title,
		headerRendered,
		m.viewport.View(),
		"",
		frameWidth,
		frameHeight,
	)

	return content
}
