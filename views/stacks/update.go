package stacksview

import (
	"fmt"
	"strings"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (view.View, tea.Cmd) {
	switch msg := msg.(type) {
	case Msg:
		m.SetContent(msg)
		m.Visible = true
		return m, nil

	case RefreshErrorMsg:
		m.Visible = true
		m.viewport.SetContent(fmt.Sprintf("Error refreshing stacks: %v", msg.Err))
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

func (m *Model) SetContent(msg Msg) {
	m.nodeId = msg.NodeId
	m.stacks = msg.Stacks
	m.stackCursor = 0

	if !m.ready {
		return
	}
	m.viewport.GotoTop()
	m.viewport.SetContent(m.buildContent())
	m.viewport.YOffset = 0
}

func (m *Model) buildContent() string {
	var b strings.Builder
	for i, stack := range m.stacks {
		cursor := "  "
		if i == m.stackCursor {
			cursor = "➜ "
		}
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, stack.Name))
	}

	m.ensureCursorVisible()
	return b.String()
}

// ensureCursorVisible keeps the cursor in view
func (m *Model) ensureCursorVisible() {
	h := m.viewport.Height
	if h < 1 {
		h = 1
	}

	if m.stackCursor < m.viewport.YOffset {
		m.viewport.YOffset = m.stackCursor
	} else if m.stackCursor >= m.viewport.YOffset+h {
		m.viewport.YOffset = m.stackCursor - h + 1
	}
}
