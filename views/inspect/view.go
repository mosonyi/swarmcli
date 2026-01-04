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

	frame := ui.ComputeFrameDimensions(
		width,
		m.viewport.Height,
		width,
		m.height,
		headerRendered,
		"",
	)

	// Get viewport content and truncate to fit the frame
	viewportContent := ui.TrimOrPadContentToLines(m.viewport.View(), frame.DesiredContentLines)

	// ---- Render framed box ----
	content := ui.RenderFramedBox(
		title,
		headerRendered,
		viewportContent,
		"",
		frame.FrameWidth,
	)

	return content
}
