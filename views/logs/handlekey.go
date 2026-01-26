// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package logsview

import (
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

	switch k.String() {
	case "q":
		m.Visible = false
		return nil
	case "esc":
		// If in search mode, exit search mode
		if m.mode == "search" {
			m.mode = "normal"
			return nil
		}
		// If in fullscreen, exit fullscreen instead of closing view
		if m.getFullscreen() {
			m.setFullscreen(false)
			l().Infof("[logsview] 'esc' key pressed: exiting fullscreen")
			return func() tea.Msg {
				return FullscreenToggledMsg{}
			}
		}
		// Otherwise, close the view
		m.Visible = false
		return nil
	case "/":
		m.mode = "search"
		m.searchTerm = ""
		m.searchIndex = 0
		return nil
	case "enter":
		if m.mode == "search" {
			m.highlightContent()
			if len(m.searchMatches) > 0 {
				m.searchIndex = 0
				m.scrollToMatch()
			}
			m.mode = "normal"
			return nil
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
	case "f":
		// toggle fullscreen mode
		oldFullscreen := m.getFullscreen()
		newFullscreen := !oldFullscreen
		m.setFullscreen(newFullscreen)
		l().Infof("[logsview] 'f' key pressed: fullscreen %v -> %v", oldFullscreen, newFullscreen)
		return func() tea.Msg {
			return FullscreenToggledMsg{}
		}
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
		// Only handle 'o' as a command in normal mode
		// In search mode, let it fall through to be captured as a rune
		if m.mode != "normal" {
			break
		}
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
	}

	// if in search mode, capture runes/backspace
	if m.mode == "search" {
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
	}

	return nil
}
