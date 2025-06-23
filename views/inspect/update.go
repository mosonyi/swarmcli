package inspectview

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
			m.viewport.SetContent(m.buildContent())
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

	if !m.ready {
		return
	}
	m.viewport.GotoTop()                    // reset scroll position
	m.viewport.SetContent(m.buildContent()) // now set new content
	m.viewport.YOffset = 0

	m.searchMatches = nil
	m.searchTerm = ""
	m.searchIndex = 0
	m.mode = "normal"
}

func (m *Model) highlightContent() {
	if m.searchTerm != "" {
		m.searchMatches = utils.FindAllMatches(m.inspectLines, m.searchTerm)
	}
	m.viewport.SetContent(m.buildContent())
}

func (m *Model) buildContent() string {
	if len(m.searchMatches) > 0 && m.searchTerm != "" {
		return utils.HighlightMatches(m.inspectLines, m.searchTerm)
	} else {
		return m.inspectLines
	}
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
