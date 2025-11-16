package logsview

import (
	"fmt"
	"strings"
	"swarmcli/utils"

	tea "github.com/charmbracelet/bubbletea"
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
		// append line into bounded buffer
		m.mu.Lock()
		// store line as-is (no newline); rendering will join with '\n'
		m.lines = append(m.lines, msg.Line)

		// trim if over MaxLines
		if m.MaxLines > 0 && len(m.lines) > m.MaxLines {
			// drop older lines
			start := len(m.lines) - m.MaxLines
			newBuf := make([]string, 0, m.MaxLines)
			newBuf = append(newBuf, m.lines[start:]...)
			m.lines = newBuf
		}

		// update searchMatches incrementally
		if m.searchTerm != "" && strings.Contains(strings.ToLower(msg.Line), strings.ToLower(m.searchTerm)) {
			m.searchMatches = append(m.searchMatches, len(m.lines)-1)
		}
		totalLines := len(m.lines)
		m.mu.Unlock()

		if m.ready {
			m.viewport.SetContent(m.buildContent())

			// auto-follow behavior: only keep bottom when follow=true and user is at bottom
			if m.follow {
				// if viewport thinks it's at bottom (or close), goto bottom
				if m.viewport.YOffset+m.viewport.Height >= m.viewport.TotalLineCount()-1 {
					m.viewport.GotoBottom()
				}
			}
			l().Debugf("[logsview] appended line; total=%d YOffset=%d Height=%d TotalLineCount=%d",
				totalLines, m.viewport.YOffset, m.viewport.Height, m.viewport.TotalLineCount())
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

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		if !m.ready {
			m.ready = true
		}
		// reset viewport content so the internal content height updates
		m.viewport.SetContent(m.buildContent())
		return nil

	case tea.KeyMsg:
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

	full := strings.Join(m.lines, "\n")
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
