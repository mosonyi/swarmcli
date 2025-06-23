package logsview

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	"swarmcli/utils"
	"swarmcli/views/view"
)

func (m Model) Update(msg tea.Msg) (view.View, tea.Cmd) {
	switch msg := msg.(type) {
	case Msg:
		m.SetContent(string(msg))
		m.Visible = true
		return m, nil

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		if !m.ready {
			m.ready = true
			m.viewport.SetContent(m.logLines) // Now set the content safely
		}

		return m, nil

	case tea.KeyMsg:
		return HandleKey(m, msg)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *Model) SetContent(content string) {
	m.logLines = content
	if len(m.searchMatches) > 0 && m.searchTerm != "" {
		content = utils.HighlightMatches(content, m.searchTerm)
	}

	m.searchMatches = nil
	m.searchTerm = ""
	m.searchIndex = 0
	m.mode = "normal"

	if !m.ready {
		return
	}
	m.viewport.GotoTop()           // reset scroll position
	m.viewport.SetContent(content) // now set new content
	m.viewport.YOffset = 0
}

func (m *Model) scrollToMatch() {
	if len(m.searchMatches) == 0 {
		return
	}
	matchPos := m.searchMatches[m.searchIndex]
	lines := strings.Split(m.logLines[:matchPos], "\n")
	offset := len(lines) - m.viewport.Height/2
	if offset < 0 {
		offset = 0
	}
	m.viewport.GotoTop()
	m.viewport.SetYOffset(offset)
}
