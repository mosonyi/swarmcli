package stacksview

import (
	"fmt"
	tea "github.com/charmbracelet/bubbletea"
	"strings"
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
	m.stackServices = msg.Services
	m.stackCursor = 0

	if !m.ready {
		return
	}
	m.viewport.GotoTop()
	m.viewport.SetContent(m.buildContent()) // now set new content
	m.viewport.YOffset = 0
}

func (m *Model) buildContent() string {
	var b strings.Builder
	for i, stack := range m.stackServices {
		cursor := "  "
		if i == m.stackCursor {
			cursor = "➜ "
		}
		b.WriteString(fmt.Sprintf("%s%s / %s\n", cursor, stack.StackName, stack.ServiceName))
	}
	return b.String()
}
