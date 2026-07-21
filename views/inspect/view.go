// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package inspectview

import (
	"fmt"
	"github.com/Eldara-Tech/swarmcli/ui"
)

func (m *Model) FrameTitle() string {
	title := m.Title
	if title == "" {
		title = "Inspecting"
	}
	return title
}

func (m *Model) FrameHeader() string {
	formatIndicator := "[YAML]"
	if m.Format == "raw" {
		formatIndicator = "[RAW]"
	}
	errorHint := ""
	if m.ParseError != "" {
		errorHint = " — Could not parse JSON, showing raw"
	}
	header := fmt.Sprintf("Inspecting %s%s", formatIndicator, errorHint)
	if m.searchMode {
		header = fmt.Sprintf("%s — Search: %s", header, m.SearchTerm)
	} else if m.SearchTerm != "" && len(m.searchMatches) > 0 {
		matchCount := len(m.searchMatches)
		currentMatch := m.searchIndex + 1
		header = fmt.Sprintf("%s — Found %d matches (%d/%d) • 'n'/'N' to navigate", header, matchCount, currentMatch, matchCount)
	}
	return ui.FrameHeaderStyle.Render(header)
}

func (m *Model) FrameFooter() string { return "" }

func (m *Model) FrameContent() string {
	header := m.FrameHeader()
	width := m.viewport.Width
	if width <= 0 {
		width = 80
	}
	frame := ui.ComputeFrameDimensions(width, m.viewport.Height, width, m.height, header, "")
	return ui.TrimOrPadContentToLines(m.viewport.View(), frame.DesiredContentLines)
}

func (m *Model) View() string {
	return ui.RenderViewFrame(m.FrameTitle(), m.FrameHeader(), m.FrameContent(), m.FrameFooter(), m.viewport.Width, m.viewport.Height, false)
}
