package nodesview

import (
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (view.View, tea.Cmd) {
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
			m.viewport.SetContent(m.renderNodes())
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
	m.viewport.SetContent(m.renderNodes())
	m.viewport.YOffset = 0
}

// ensureCursorVisible keeps the cursor within the visible viewport
func (m *Model) ensureCursorVisible() {
	// Prevent negative height
	h := m.viewport.Height
	if h < 1 {
		h = 1
	}

	// If cursor is above the viewport, scroll up
	if m.cursor < m.viewport.YOffset {
		m.viewport.YOffset = m.cursor
	} else if m.cursor >= m.viewport.YOffset+h {
		// If cursor is below viewport, scroll down
		m.viewport.YOffset = m.cursor - h + 1
	}
}
