package logsview

import (
	tea "github.com/charmbracelet/bubbletea"
)

func HandleKey(m *Model, k tea.KeyMsg) tea.Cmd {
	switch k.String() {
	case "q", "esc":
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
	case "f":
		// toggle follow mode
		oldFollow := m.getFollow()
		newFollow := !oldFollow
		m.setFollow(newFollow)
		l().Infof("[logsview] 'f' key pressed: follow %v -> %v", oldFollow, newFollow)
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
