// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package logsview

import (
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

func HandleKey(m *Model, k tea.KeyMsg) tea.Cmd {
	// Handle node selection dialog first if visible
	if m.getNodeSelectVisible() {
		switch k.String() {
		case "esc":
			m.setNodeSelectVisible(false)
			return nil
		case "up", "k":
			m.mu.Lock()
			if m.nodeSelectCursor > 0 {
				m.nodeSelectCursor--
			}
			m.mu.Unlock()
			return nil
		case "down", "j":
			m.mu.Lock()
			if m.nodeSelectCursor < len(m.nodeSelectNodes)-1 {
				m.nodeSelectCursor++
			}
			m.mu.Unlock()
			return nil
		case "pgup":
			// Jump up by 5 items
			m.mu.Lock()
			m.nodeSelectCursor -= 5
			if m.nodeSelectCursor < 0 {
				m.nodeSelectCursor = 0
			}
			m.mu.Unlock()
			return nil
		case "pgdown":
			// Jump down by 5 items
			m.mu.Lock()
			m.nodeSelectCursor += 5
			if m.nodeSelectCursor >= len(m.nodeSelectNodes) {
				m.nodeSelectCursor = len(m.nodeSelectNodes) - 1
			}
			m.mu.Unlock()
			return nil
		case "enter":
			m.mu.Lock()
			// Safety check: ensure cursor is within bounds
			if m.nodeSelectCursor < 0 || m.nodeSelectCursor >= len(m.nodeSelectNodes) || len(m.nodeSelectNodes) == 0 {
				m.mu.Unlock()
				m.setNodeSelectVisible(false)
				return nil
			}
			selectedNode := m.nodeSelectNodes[m.nodeSelectCursor]
			m.mu.Unlock()

			if selectedNode == "All nodes" {
				m.setNodeFilter("")
			} else {
				m.setNodeFilter(selectedNode)
			}
			m.setNodeSelectVisible(false)
			l().Infof("[logsview] Selected node filter: %q", m.getNodeFilter())
			return func() tea.Msg {
				return NodeFilterToggledMsg{}
			}
		}
		return nil
	}

	// In search mode, only handle special keys and text input
	if m.mode == "search" {
		switch k.String() {
		case "esc":
			// Exit search mode
			m.mode = "normal"
			return nil
		case "enter":
			// Perform search and exit search mode
			m.highlightContent()
			if len(m.searchMatches) > 0 {
				m.searchIndex = 0
				m.scrollToMatch()
			}
			m.mode = "normal"
			return nil
		case " ", "space":
			// Handle space key explicitly
			m.searchTerm += " "
			m.highlightContent()
			return nil
		}
		// In search mode, capture runes/backspace as text input
		switch k.Type {
		case tea.KeyRunes:
			m.searchTerm += string(k.Runes)
			m.highlightContent()
			return nil
		case tea.KeyBackspace:
			if len(m.searchTerm) > 0 {
				m.searchTerm = m.searchTerm[:len(m.searchTerm)-1]
				m.highlightContent()
			}
			return nil
		}
		// For other keys in search mode, ignore them (don't let them fall through)
		return nil
	}

	// Normal mode key handling
	switch k.String() {
	case "esc":
		// If app-level filter is active, clear it instead of closing
		if m.getFilterQuery() != "" {
			m.mu.Lock()
			m.filterQuery = ""
			m.mu.Unlock()
			m.highlightContent()
			return nil
		}
		// Close the view
		m.Visible = false
		return nil
	case "ctrl+f":
		m.mode = "search"
		m.searchTerm = ""
		m.searchIndex = 0
		return nil
	case "enter":
		// The bash gesture: hitting enter under a `tail -f` pushes what has
		// been read up the screen, so whatever arrives next is unmistakably
		// new. There is no scrollback to push here, so the break goes into the
		// buffer instead — at the end, under everything read so far.
		m.mu.Lock()
		m.appendLine(time.Now().Format(markTimeFormat), "", "", lineMark, time.Time{})
		m.trimToMaxLines()
		m.mu.Unlock()
		l().Debugf("[logsview] 'enter' key pressed: separator inserted")
		return func() tea.Msg {
			return MarkInsertedMsg{}
		}
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
	case "s":
		// toggle follow mode
		oldFollow := m.getFollow()
		newFollow := !oldFollow
		m.setFollow(newFollow)
		l().Infof("[logsview] 's' key pressed: follow %v -> %v", oldFollow, newFollow)
		return nil
	case "w":
		// toggle wrap mode
		oldWrap := m.getWrap()
		newWrap := !oldWrap
		m.setWrap(newWrap)
		// Reset horizontal offset when enabling wrap
		if newWrap {
			m.horizontalOffset = 0
		}
		l().Infof("[logsview] 'w' key pressed: wrap %v -> %v", oldWrap, newWrap)
		// Refresh content with new wrap setting
		return func() tea.Msg {
			return WrapToggledMsg{}
		}
	case "left", "h":
		// Scroll left when wrap is off
		if !m.getWrap() {
			if m.horizontalOffset > 0 {
				m.horizontalOffset -= 10 // Scroll by 10 characters
				if m.horizontalOffset < 0 {
					m.horizontalOffset = 0
				}
				return func() tea.Msg {
					return WrapToggledMsg{} // Reuse to refresh content
				}
			}
		}
		return nil
	case "right", "l":
		// Scroll right when wrap is off
		if !m.getWrap() {
			// Calculate max line length to determine scroll limit
			m.mu.Lock()
			maxLen := 0
			for _, line := range m.lines {
				if len(line) > maxLen {
					maxLen = len(line)
				}
			}
			m.mu.Unlock()

			// Calculate max scroll: stop when the end of the longest line is at screen center
			maxScroll := maxLen - (m.viewport.Width / 2)
			if maxScroll < 0 {
				maxScroll = 0
			}

			// Only scroll if we haven't reached the limit
			if m.horizontalOffset < maxScroll {
				m.horizontalOffset += 10 // Scroll by 10 characters
				// Cap at max scroll position
				if m.horizontalOffset > maxScroll {
					m.horizontalOffset = maxScroll
				}
				return func() tea.Msg {
					return WrapToggledMsg{} // Reuse to refresh content
				}
			}
		}
		return nil
	case "o":
		// Show node selection dialog
		nodes := m.extractUniqueNodes()
		if len(nodes) > 1 { // More than just "All nodes"
			m.mu.Lock()
			m.nodeSelectVisible = true
			m.nodeSelectNodes = nodes
			m.nodeSelectCursor = 0
			// Set cursor to current filter if exists
			currentFilter := m.nodeFilter
			m.mu.Unlock()

			if currentFilter != "" {
				m.mu.Lock()
				for i, node := range nodes {
					if node == currentFilter {
						m.nodeSelectCursor = i
						break
					}
				}
				m.mu.Unlock()
			}
			l().Infof("[logsview] 'o' key pressed: showing node selection dialog with %d nodes", len(nodes))
		}
		return nil
	case "t":
		// Toggle showing logs from stopped (non-running) tasks
		old := m.getHideStopped()
		m.setHideStopped(!old)
		l().Infof("[logsview] 't' key pressed: hideStopped %v -> %v", old, !old)
		return func() tea.Msg {
			return HideStoppedToggledMsg{}
		}
	}

	return nil
}
