package nodesview

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case Msg:
		m.SetContent(msg)
		m.Visible = true
		return nil

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		if !m.ready {
			m.ready = true
			m.viewport.SetContent(m.renderNodes())
		}
		return nil

	case tea.KeyMsg:
		return HandleKey(m, msg)
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return cmd
}

func (m *Model) SetContent(msg Msg) {
	m.entries = msg.Entries
	m.cursor = 0

	if !m.ready {
		return
	}

	m.viewport.GotoTop()
	m.viewport.SetContent(m.renderNodes())
	m.viewport.YOffset = 0
}
