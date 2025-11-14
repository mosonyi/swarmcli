package logsview

import (
	"fmt"
	"strings"
	"swarmcli/utils"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (view.View, tea.Cmd) {
	switch msg := msg.(type) {

	case InitStreamMsg:
		m.linesChan = msg.Lines
		m.errChan = msg.Errs
		m.Visible = true
		return m, m.readOneLineCmd()

	case LineMsg:
		m.mu.Lock()
		line := msg.Line
		if strings.HasSuffix(line, "\n") {
			line = line[:len(line)-1]
		}
		m.lines = append(m.lines, line)

		if m.searchTerm != "" && strings.Contains(strings.ToLower(line), strings.ToLower(m.searchTerm)) {
			m.searchMatches = append(m.searchMatches, len(m.lines)-1)
		}
		m.mu.Unlock()

		if m.ready {
			m.viewport.SetContent(m.buildContent())

			// auto-scroll only if user is at bottom
			if m.viewport.YOffset+m.viewport.Height >= m.viewport.TotalLineCount()-1 {
				m.viewport.GotoBottom()
			}
		}

		return m, m.readOneLineCmd()

	case StreamErrMsg:
		m.mu.Lock()
		m.lines = append(m.lines, fmt.Sprintf("Error: %v", msg.Err))
		m.mu.Unlock()

		if m.ready {
			m.viewport.SetContent(m.buildContent())
		}
		return m, nil

	case StreamDoneMsg:
		m.mu.Lock()
		m.lines = append(m.lines, "--- stream closed ---")
		m.mu.Unlock()

		if m.ready {
			m.viewport.SetContent(m.buildContent())
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		if !m.ready {
			m.ready = true
		}

		m.viewport.SetContent(m.buildContent())
		return m, nil

	case tea.KeyMsg:

		// -----------------------------------------
		// 1) Let viewport handle all scrolling keys
		// -----------------------------------------
		switch msg.String() {
		case "up", "down", "pgup", "pgdown", "home", "end":
			var cmd tea.Cmd
			m.viewport, cmd = m.viewport.Update(msg)
			return m, cmd
		}

		// -----------------------------------------------------
		// 2) Other keys → handled by your custom handler
		// -----------------------------------------------------
		newModel, cmd := HandleKey(m, msg)
		return newModel, cmd
	}

	// default: feed remaining msgs to viewport
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

// readOneLineCmd returns a cmd that blocks until one line is available from linesChan.
// It returns a LineMsg, or StreamErrMsg/StreamDoneMsg when the stream closes/errors.
func (m Model) readOneLineCmd() tea.Cmd {
	// if channels are not set, no-op
	if m.linesChan == nil && m.errChan == nil {
		return nil
	}

	return func() tea.Msg {
		// note: we don't hold any locks here - the producer goroutine writes to these channels
		// wait for either a line, an error, or closed channel
		select {
		case line, ok := <-m.linesChan:
			if !ok {
				// no more lines -> done
				return StreamDoneMsg{}
			}
			return LineMsg{Line: line}
		case err, ok := <-m.errChan:
			if !ok {
				// errs channel closed and no error
				// but we still should check linesChan in next invocation
				return StreamDoneMsg{}
			}
			if err != nil {
				return StreamErrMsg{Err: err}
			}
			return StreamDoneMsg{}
		}
	}
}

// SetContent replaces entire content (keeps compatibility with previous API).
// It resets search state.
func (m *Model) SetContent(content string) {
	m.mu.Lock()
	m.lines = strings.Split(content, "\n")
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
}

// highlightContent recalculates matches and rebuilds visible content.
func (m *Model) highlightContent() {
	if m.searchTerm == "" {
		m.searchMatches = nil
	} else {
		// recalc matches across all lines
		m.searchMatches = []int{}
		lowerTerm := strings.ToLower(m.searchTerm)
		m.mu.Lock()
		for i, L := range m.lines {
			if strings.Contains(strings.ToLower(L), lowerTerm) {
				m.searchMatches = append(m.searchMatches, i)
			}
		}
		m.mu.Unlock()
		if len(m.searchMatches) > 0 {
			// clamp index
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

// buildContent constructs the string shown in the viewport. It only formats the visible slice.
func (m *Model) buildContent() string {
	m.mu.Lock()
	defer m.mu.Unlock()

	// determine visible lines from viewport offset & height
	start := m.viewport.YOffset
	if start < 0 {
		start = 0
	}
	end := start + m.viewport.Height
	if end > len(m.lines) {
		end = len(m.lines)
	}

	visible := strings.Join(m.lines[start:end], "\n")
	// if we're in search mode and have a term, highlight matches within visible text
	if m.mode == "search" && m.searchTerm != "" {
		return utils.HighlightMatches(visible, m.searchTerm)
	}
	return visible
}

// scrollToMatch makes the viewport scroll to the currently selected match.
func (m *Model) scrollToMatch() {
	if len(m.searchMatches) == 0 || m.mode != "search" {
		return
	}
	// get the line index for current match
	idx := m.searchMatches[m.searchIndex]
	// center that line within viewport
	offset := idx - m.viewport.Height/2
	if offset < 0 {
		offset = 0
	}
	m.viewport.GotoTop()
	m.viewport.SetYOffset(offset)
	// after moving, update visible content
	m.viewport.SetContent(m.buildContent())
}
