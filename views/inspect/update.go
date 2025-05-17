package inspectview

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case Msg:
		m.inspectLines = string(msg)
		m.viewport.SetContent(m.inspectLines)
		m.Visible = true

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height - 2
		return m, nil

	case tea.KeyMsg:
		return HandleKey(m, msg)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *Model) SetContent(content string) {
	m.viewport.SetContent(content)
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
