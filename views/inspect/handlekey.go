// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package inspectview

import (
	tea "github.com/charmbracelet/bubbletea"
)

func handleNormalKey(m *Model, k tea.KeyMsg) tea.Cmd {
	switch k.String() {
	case "q", "esc":
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

	case "/", "shift+/":
		m.searchMode = true
		m.SearchTerm = ""
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
		m.updateViewport()
	case tea.KeyEsc:
		m.searchMode = false
		m.SearchTerm = ""
		m.updateViewport()
	}
	return nil
}
