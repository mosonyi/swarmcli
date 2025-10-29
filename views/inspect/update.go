package inspectview

import (
	"strings"
	"swarmcli/utils"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
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

func (m *Model) SetContent(jsonStr string) {
	root, err := ParseJSON(jsonStr)
	if err != nil {
		m.inspectLines = "Error parsing JSON: " + err.Error()
		m.inspectRoot = nil
	} else {
		m.inspectRoot = root
		m.inspectLines = strings.Join(RenderTree(root, 0, m.expanded), "\n")
	}

	if !m.ready {
		return
	}
	m.viewport.GotoTop()
	m.viewport.SetContent(m.buildContent())

	m.searchMatches = nil
	m.searchTerm = ""
	m.searchIndex = 0
	m.mode = "normal"
}

func (m *Model) buildContent() string {
	if m.searchTerm != "" && len(m.searchMatches) > 0 {
		return utils.HighlightMatches(m.inspectLines, m.searchTerm)
	}
	return m.inspectLines
}
