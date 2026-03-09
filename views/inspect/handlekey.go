// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package inspectview

import (
	tea "github.com/charmbracelet/bubbletea"
)

func handleNormalKey(m *Model, k tea.KeyMsg) tea.Cmd {
	switch k.String() {
	case "q":
		return nil
	case "esc":
		// If app-level filter is active, clear it instead of closing
		if m.filterQuery != "" {
			m.filterQuery = ""
			m.updateViewport()
			return nil
		}
		return nil
	case "up", "k":
		m.viewport.ScrollUp(1)
	case "down", "j":
		m.viewport.ScrollDown(1)
	case "pgup":
		m.viewport.ScrollUp(m.viewport.Height)
	case "pgdown":
		m.viewport.ScrollDown(m.viewport.Height)
	case "r":
		if m.Format == "raw" {
			m.SetFormat("yml")
		} else {
			m.SetFormat("raw")
		}
		return nil

	case "ctrl+f":
		m.searchMode = true
		m.SearchTerm = ""
		return nil

	case "n":
		if len(m.searchMatches) > 0 {
			m.searchIndex = (m.searchIndex + 1) % len(m.searchMatches)
			m.scrollToMatch()
		}
		return nil
	case "N":
		if len(m.searchMatches) > 0 {
			m.searchIndex = (m.searchIndex - 1 + len(m.searchMatches)) % len(m.searchMatches)
			m.scrollToMatch()
		}
		return nil
	}
	return nil
}

func handleSearchKey(m *Model, k tea.KeyMsg) tea.Cmd {
	switch k.Type {
	case tea.KeyRunes:
		m.SearchTerm += k.String()
		m.updateViewport()
	case tea.KeyBackspace:
		if len(m.SearchTerm) > 0 {
			m.SearchTerm = m.SearchTerm[:len(m.SearchTerm)-1]
			m.updateViewport()
		}
	case tea.KeyEnter:
		m.searchMode = false
		m.computeSearchMatches()
		if len(m.searchMatches) > 0 {
			m.searchIndex = 0
			m.scrollToMatch()
		} else {
			m.updateViewport()
		}
	case tea.KeyEsc:
		m.searchMode = false
		m.SearchTerm = ""
		m.searchMatches = nil
		m.searchIndex = 0
		m.updateViewport()
	}
	return nil
}
