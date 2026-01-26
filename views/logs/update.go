// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package logsview

import (
	"fmt"
	"strings"
	"swarmcli/utils"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/muesli/reflow/wordwrap"
)

// Update processes Tea messages.
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {

	case InitStreamMsg:
		// store channels and begin the read-once pump
		m.linesChan = msg.Lines
		m.errChan = msg.Errs
		m.Visible = true
		l().Debugf("[logsview] stream initialized")
		return m.readOneLineCmd()

	case LineMsg:
		// Parse node name from the line (format: "nodename\x00actual_line")
		parts := strings.SplitN(msg.Line, "\x00", 2)
		var nodeName, actualLine string
		if len(parts) == 2 {
			nodeName = parts[0]
			actualLine = parts[1]
		} else {
			actualLine = msg.Line
		}

		// append line into bounded buffer (store both line and node)
		m.mu.Lock()
		// store line as-is (no newline); rendering will join with '\n'
		m.lines = append(m.lines, actualLine)
		m.lineNodes = append(m.lineNodes, nodeName)

		// track how many lines we're dropping from the top
		linesDropped := 0

		// trim if over MaxLines
		if m.MaxLines > 0 && len(m.lines) > m.MaxLines {
			// drop older lines from both slices
			start := len(m.lines) - m.MaxLines
			linesDropped = start
			newBuf := make([]string, 0, m.MaxLines)
			newBuf = append(newBuf, m.lines[start:]...)
			m.lines = newBuf

			newNodeBuf := make([]string, 0, m.MaxLines)
			newNodeBuf = append(newNodeBuf, m.lineNodes[start:]...)
			m.lineNodes = newNodeBuf
		}

		// update searchMatches incrementally
		if m.searchTerm != "" && strings.Contains(strings.ToLower(actualLine), strings.ToLower(m.searchTerm)) {
			m.searchMatches = append(m.searchMatches, len(m.lines)-1)
		}
		totalLines := len(m.lines)
		shouldFollow := m.follow
		m.mu.Unlock()

		if m.ready {
			// auto-follow behavior: only scroll to bottom when follow is enabled
			if shouldFollow {
				m.viewport.SetContent(m.buildContent())
				m.viewport.GotoBottom()
				l().Debugf("[logsview] auto-scrolled to bottom (follow=true)")
			} else {
				// Save current offset before updating content
				savedOffset := m.viewport.YOffset
				m.viewport.SetContent(m.buildContent())

				// Adjust offset if we dropped lines from the top
				newOffset := savedOffset
				if linesDropped > 0 {
					newOffset = savedOffset - linesDropped
					if newOffset < 0 {
						newOffset = 0
					}
				}

				// Ensure offset is within bounds (important when wrapping changes line count)
				maxOffset := m.viewport.TotalLineCount() - m.viewport.Height
				if maxOffset < 0 {
					maxOffset = 0
				}
				if newOffset > maxOffset {
					newOffset = maxOffset
				}

				m.viewport.YOffset = newOffset
				l().Debugf("[logsview] NOT scrolling (follow=false), YOffset=%d->%d (dropped %d lines)", savedOffset, newOffset, linesDropped)
			}
			l().Debugf("[logsview] appended line; total=%d YOffset=%d Height=%d TotalLineCount=%d follow=%v",
				totalLines, m.viewport.YOffset, m.viewport.Height, m.viewport.TotalLineCount(), shouldFollow)
		}
		return m.readOneLineCmd()

	case StreamErrMsg:
		// append an error line and stop
		m.mu.Lock()
		m.lines = append(m.lines, fmt.Sprintf("Error: %v", msg.Err))
		m.mu.Unlock()
		l().Errorf("[logsview] stream error: %v", msg.Err)
		if m.ready {
			m.viewport.SetContent(m.buildContent())
		}
		return nil

	case StreamDoneMsg:
		m.mu.Lock()
		m.lines = append(m.lines, "--- stream closed ---")
		m.mu.Unlock()
		l().Debugf("[logsview] stream closed")
		if m.ready {
			m.viewport.SetContent(m.buildContent())
		}
		return nil

	case WrapToggledMsg:
		// Refresh viewport content with new wrap setting
		if m.ready {
			// Reset to a safe position when toggling wrap
			// because line count changes dramatically with wrapping
			savedOffset := m.viewport.YOffset
			m.viewport.SetContent(m.buildContent())

			// Ensure YOffset is within bounds
			maxOffset := m.viewport.TotalLineCount() - m.viewport.Height
			if maxOffset < 0 {
				maxOffset = 0
			}

			// If we were following, go to bottom
			shouldFollow := m.getFollow()
			if shouldFollow {
				m.viewport.GotoBottom()
			} else if savedOffset > maxOffset {
				// Adjust offset to stay within bounds
				m.viewport.YOffset = maxOffset
			} else {
				m.viewport.YOffset = savedOffset
			}
		}
		return nil

	case FullscreenToggledMsg:
		// Fullscreen mode is handled in the View() method
		// Just trigger a re-render
		return nil

	case NodeFilterToggledMsg:
		// When node filter changes, we need to rebuild the content
		// because existing lines need to be filtered/unfiltered
		if m.ready {
			m.viewport.SetContent(m.buildContent())
			if m.getFollow() {
				m.viewport.GotoBottom()
			}
		}
		return nil

	case tea.WindowSizeMsg:
		// Safety check: ensure dimensions are positive
		if msg.Width < 1 {
			msg.Width = 1
		}
		if msg.Height < 1 {
			msg.Height = 1
		}

		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		if !m.ready {
			m.ready = true
		}
		// reset viewport content so the internal content height updates
		m.viewport.SetContent(m.buildContent())
		return nil

	case tea.KeyMsg:
		// Check if node select dialog is visible first
		if m.getNodeSelectVisible() {
			// When dialog is visible, handle ALL keys through HandleKey
			// to prevent any keys from falling through to the viewport
			cmd := HandleKey(m, msg)
			// Force a return here to prevent any further processing
			if cmd != nil {
				return cmd
			}
			// Even if cmd is nil, don't process the key further
			return nil
		}

		// 1) allow viewport to handle scrolling keys
		switch msg.String() {
		case "up", "down", "pgup", "pgdown", "home", "end", "k", "j":
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return cmd
		}

		// 2) other keys -> our handler
		cmd := HandleKey(m, msg)
		return cmd
	}

	// default: let viewport handle other messages
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return cmd
}

// readOneLineCmd returns a cmd that waits for one line from the line channel.
func (m *Model) readOneLineCmd() tea.Cmd {
	if m.linesChan == nil && m.errChan == nil {
		return nil
	}
	return func() tea.Msg {
		select {
		case line, ok := <-m.linesChan:
			if !ok {
				return StreamDoneMsg{}
			}
			return LineMsg{Line: line}
		case err, ok := <-m.errChan:
			if !ok {
				return StreamDoneMsg{}
			}
			if err != nil {
				return StreamErrMsg{Err: err}
			}
			return StreamDoneMsg{}
		}
	}
}

func (m *Model) SetContent(content string) {
	m.mu.Lock()
	m.lines = strings.Split(content, "\n")
	if m.MaxLines > 0 && len(m.lines) > m.MaxLines {
		// keep only last MaxLines
		start := len(m.lines) - m.MaxLines
		m.lines = append([]string{}, m.lines[start:]...)
	}
	m.searchMatches = nil
	m.searchTerm = ""
	m.searchIndex = 0
	m.mode = "normal"
	m.mu.Unlock()

	if !m.ready {
		return
	}
	m.viewport.GotoTop()
	m.viewport.SetContent(m.buildContent())
	m.viewport.YOffset = 0
	l().Debugf("[logsview] SetContent called: total lines=%d", len(m.lines))
}

func (m *Model) highlightContent() {
	if m.searchTerm == "" {
		m.searchMatches = nil
	} else {
		m.searchMatches = []int{}
		lower := strings.ToLower(m.searchTerm)
		m.mu.Lock()
		for i, L := range m.lines {
			if strings.Contains(strings.ToLower(L), lower) {
				m.searchMatches = append(m.searchMatches, i)
			}
		}
		m.mu.Unlock()
		if len(m.searchMatches) > 0 {
			if m.searchIndex >= len(m.searchMatches) {
				m.searchIndex = 0
			}
		} else {
			m.searchIndex = 0
		}
	}
	if m.ready {
		m.viewport.SetContent(m.buildContent())
	}
}

// buildContent returns the full content (required by viewport) — HighlightMatches may return colored output.
func (m *Model) buildContent() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Apply node filter
	nodeFilter := m.nodeFilter
	var filteredLines []string
	if nodeFilter != "" {
		// Filter lines by node
		for i, line := range m.lines {
			if i < len(m.lineNodes) && m.lineNodes[i] == nodeFilter {
				filteredLines = append(filteredLines, line)
			}
		}
	} else {
		// No filter, use all lines
		filteredLines = m.lines
	}

	// Join lines first
	full := strings.Join(filteredLines, "\n")

	// Apply wrapping based on wrap setting
	// BUT: skip wrapping if node selection dialog is visible to avoid overlay issues
	if m.wrap && m.viewport.Width > 0 && !m.nodeSelectVisible {
		// Wrap the entire content to viewport width
		full = wordwrap.String(full, m.viewport.Width)
	} else if (!m.wrap || m.nodeSelectVisible) && m.viewport.Width > 0 {
		// When wrap is off, apply horizontal scrolling
		processedLines := make([]string, len(filteredLines))

		for i, line := range filteredLines {
			if len(line) <= m.horizontalOffset {
				// Line is shorter than offset, show empty
				processedLines[i] = ""
			} else {
				// Apply horizontal offset
				visiblePart := line[m.horizontalOffset:]

				if len(visiblePart) > m.viewport.Width {
					// Truncate and add > indicator
					if m.viewport.Width > 1 {
						processedLines[i] = visiblePart[:m.viewport.Width-1] + ">"
					} else {
						processedLines[i] = ">"
					}
				} else {
					processedLines[i] = visiblePart
				}
			}
		}
		full = strings.Join(processedLines, "\n")
	}

	if m.mode == "search" && m.searchTerm != "" {
		return utils.HighlightMatches(full, m.searchTerm)
	}
	return full
}

// scrollToMatch centers the viewport on the selected match
func (m *Model) scrollToMatch() {
	if len(m.searchMatches) == 0 || m.mode != "search" {
		return
	}
	idx := m.searchMatches[m.searchIndex]
	offset := idx - m.viewport.Height/2
	if offset < 0 {
		offset = 0
	}
	m.viewport.GotoTop()
	m.viewport.SetYOffset(offset)
	m.viewport.SetContent(m.buildContent())
	l().Debugf("[logsview] scrollToMatch idx=%d newYOffset=%d", idx, offset)
}
