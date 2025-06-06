package inspectview

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	"swarmcli/utils"
)

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {

	switch msg := msg.(type) {
	case Msg:
		if m.ready {
			m.SetContent(string(msg))
		}
		m.Visible = true
		return m, nil

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		if !m.ready {
			m.ready = true
			m.viewport.SetContent(m.inspectLines) // Now set the content safely
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
	m.inspectLines = content
	if len(m.searchMatches) > 0 && m.searchTerm != "" {
		content = utils.HighlightMatches(content, m.searchTerm)
	}

	if !m.ready {
		return
	}
	m.viewport.GotoTop()           // reset scroll position
	m.viewport.SetContent(content) // now set new content
	m.viewport.YOffset = 0

	m.searchMatches = nil
	m.searchTerm = ""
	m.searchIndex = 0
	m.mode = "normal"
}

func (m *Model) scrollToMatch() {
	if len(m.searchMatches) == 0 {
		return
	}
	matchPos := m.searchMatches[m.searchIndex]
	lines := strings.Split(m.inspectLines[:matchPos], "\n")
	offset := len(lines) - m.viewport.Height/2
	if offset < 0 {
		offset = 0
	}
	m.viewport.GotoTop()
	m.viewport.SetYOffset(offset)
}
