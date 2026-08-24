// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package logsview

import (
	"fmt"
	"github.com/Eldara-Tech/swarmcli/v2/ui"
	"github.com/Eldara-Tech/swarmcli/v2/utils"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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
		// Parse node name and task ID (format: "nodename\x00taskid\x00actual_line")
		parts := strings.SplitN(msg.Line, "\x00", 3)
		var nodeName, taskID, actualLine string
		if len(parts) == 3 {
			nodeName = parts[0]
			taskID = parts[1]
			actualLine = parts[2]
		} else {
			actualLine = msg.Line
		}

		// Strip carriage returns that break frame rendering (progress bars, CRLF)
		actualLine = strings.ReplaceAll(actualLine, "\r", "")

		// append line into bounded buffer (store line, node and task)
		m.mu.Lock()
		// store line as-is (no newline); rendering will join with '\n'
		m.lines = append(m.lines, actualLine)
		m.lineNodes = append(m.lineNodes, nodeName)
		m.lineTasks = append(m.lineTasks, taskID)

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

			newTaskBuf := make([]string, 0, m.MaxLines)
			newTaskBuf = append(newTaskBuf, m.lineTasks[start:]...)
			m.lineTasks = newTaskBuf
		}

		shouldFollow := m.follow
		m.mu.Unlock()

		if m.ready {
			// auto-follow behavior: only scroll to bottom when follow is enabled
			if shouldFollow {
				m.viewport.SetContent(m.buildContent())
				m.viewport.GotoBottom()
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
			}
		}
		return m.readOneLineCmd()

	case StreamErrMsg:
		// append an error line and stop
		m.mu.Lock()
		m.lines = append(m.lines, fmt.Sprintf("Error: %v", msg.Err))
		m.lineNodes = append(m.lineNodes, "")
		m.lineTasks = append(m.lineTasks, "")
		m.mu.Unlock()
		l().Errorf("[logsview] stream error: %v", msg.Err)
		if m.ready {
			m.viewport.SetContent(m.buildContent())
		}
		return nil

	case StreamDoneMsg:
		m.mu.Lock()
		m.lines = append(m.lines, "--- stream closed ---")
		m.lineNodes = append(m.lineNodes, "")
		m.lineTasks = append(m.lineTasks, "")
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

	case HideStoppedToggledMsg:
		// The visible set changed; recompute search-match indices (this also
		// rebuilds the viewport content) and re-anchor to the bottom if following.
		if m.ready {
			m.highlightContent()
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
		// msg.Height is the frame's height; the viewport only gets what is left
		// after the frame's own rows and the header. Sizing it to the frame
		// instead would park the newest lines below the cut — GotoBottom would
		// anchor a window taller than the rows actually drawn.
		m.viewport.Height = max(1, ui.ContentRows(msg.Height, ui.FramedChromeRows, m.FrameHeader(), m.FrameFooter()))
		if !m.ready {
			m.ready = true
		}
		// reset viewport content so the internal content height updates
		m.viewport.SetContent(m.buildContent())
		// A resize changes how many lines fit, but not the offset the viewport
		// scrolls from, so it keeps drawing the lines it drew before: growing
		// leaves the rows it gained blank, shrinking cuts the newest lines off.
		// Re-anchor while following, and otherwise only when the offset now sits
		// past the end — a reader who scrolled up stays where they were.
		if m.getFollow() || m.viewport.PastBottom() {
			m.viewport.GotoBottom()
		}
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

		// In search mode, don't intercept any keys for viewport - let HandleKey process them all
		if m.mode == "search" {
			cmd := HandleKey(m, msg)
			return cmd
		}

		// 1) allow viewport to handle scrolling keys (only in normal mode)
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
	content = strings.ReplaceAll(content, "\r", "")
	m.lines = strings.Split(content, "\n")
	if m.MaxLines > 0 && len(m.lines) > m.MaxLines {
		// keep only last MaxLines
		start := len(m.lines) - m.MaxLines
		m.lines = append([]string{}, m.lines[start:]...)
	}
	// SetContent carries no per-line node/task metadata; reset the parallel
	// slices to empty (length-aligned) so all lines are treated as
	// unfilterable (always visible) and indices stay aligned.
	m.lineNodes = make([]string, len(m.lines))
	m.lineTasks = make([]string, len(m.lines))
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

// highlightContent rebuilds the rendered content, which is also what recomputes
// the search matches, and re-anchors the selected match inside the new set.
func (m *Model) highlightContent() {
	content := m.buildContent()
	if m.searchIndex >= len(m.searchMatches) {
		m.searchIndex = 0
	}
	if m.ready {
		m.viewport.SetContent(content)
	}
}

// buildContent returns the full content (required by viewport) — HighlightMatches may return colored output.
func (m *Model) buildContent() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Apply all active filters (node, "/" text, hide-stopped) in a single pass
	// via the shared predicate so the visible set matches highlightContent.
	var stopped map[string]bool
	if m.hideStopped {
		stopped = m.stoppedTaskIDs()
	}
	// Collect the search matches in the same pass. They are indices into the
	// visible lines, so they can only be counted where the visible set is
	// built — deriving them anywhere else means a second copy of this walk,
	// and the two drifting apart is what froze the match counter (#586).
	lower := strings.ToLower(m.searchTerm)
	m.searchMatches = nil
	var filteredLines []string
	for i, line := range m.lines {
		if !m.lineVisible(i, stopped) {
			continue
		}
		if lower != "" && strings.Contains(strings.ToLower(line), lower) {
			m.searchMatches = append(m.searchMatches, len(filteredLines))
		}
		filteredLines = append(filteredLines, line)
	}
	m.visibleCount = len(filteredLines)

	// Join lines first
	full := strings.Join(filteredLines, "\n")

	// Apply wrapping based on wrap setting
	// BUT: skip wrapping if node selection dialog is visible to avoid overlay issues
	if m.wrap && m.viewport.Width > 0 && !m.nodeSelectVisible {
		// Wrap the entire content to viewport width
		full = wordwrap.String(full, m.viewport.Width)
	} else if (!m.wrap || m.nodeSelectVisible) && m.viewport.Width > 0 {
		// When wrap is off, apply horizontal scrolling using ANSI-aware operations
		// to avoid splitting escape sequences in colored log lines.
		processedLines := make([]string, len(filteredLines))

		for i, line := range filteredLines {
			lineWidth := lipgloss.Width(line)
			if lineWidth <= m.horizontalOffset {
				processedLines[i] = ""
			} else {
				visiblePart := ui.TruncateANSIAfter(line, m.horizontalOffset)
				visibleWidth := lipgloss.Width(visiblePart)

				if visibleWidth > m.viewport.Width {
					if m.viewport.Width > 1 {
						processedLines[i] = ui.TruncateANSI(visiblePart, m.viewport.Width-1) + ">"
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

	// Apply highlighting if we have an active search, regardless of mode
	if m.searchTerm != "" {
		return utils.HighlightMatches(full, m.searchTerm)
	}
	return full
}

// scrollToMatch centers the viewport on the selected match
func (m *Model) scrollToMatch() {
	if len(m.searchMatches) == 0 {
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
