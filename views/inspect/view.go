// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package inspectview

import (
	"fmt"
	"github.com/Eldara-Tech/swarmcli/v2/ui"
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

// FrameContent returns the viewport as-is: it is already sized to the rows the
// frame will draw.
func (m *Model) FrameContent() string { return m.viewport.View() }

// View renders the view on its own; the app composes it from the Frame* parts
// instead. The box is drawn around the content rather than to a height, since
// the viewport is already sized to the rows it should fill.
func (m *Model) View() string {
	return ui.RenderFramedBox(m.FrameTitle(), m.FrameHeader(), m.FrameContent(), m.FrameFooter(),
		m.viewport.Width+ui.FrameChromeColumns)
}
