package nodesview

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case Msg:
		m.SetContent(msg)
		m.Visible = true
		return m, nil

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		if !m.ready {
			m.ready = true
			m.viewport.SetContent(m.buildContent()) // Now set the content safely
		}

		return m, nil

	case tea.KeyMsg:
		return HandleKey(m, msg)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}

func (m *Model) SetContent(msg Msg) {
	m.nodes = msg
	m.cursor = 0

	if !m.ready {
		return
	}
	m.viewport.GotoTop()
	m.viewport.SetContent(m.buildContent()) // now set new content
	m.viewport.YOffset = 0
}

func (m *Model) buildContent() string {
	var s string
	for i, item := range m.nodes {
		cursor := "  "
		if i == m.cursor {
			cursor = "→ "
		}
		s += fmt.Sprintf("%s%s\n", cursor, item)
	}
	return s
}
