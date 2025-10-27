package stacksview

import (
	"fmt"
	"strings"
	"swarmcli/docker"
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
	m.nodeId = msg.NodeId
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
	visible := m.visibleStackServices()
	start := m.stackCursor - m.stackCursor%m.viewport.Height
	for i, stack := range visible {
		cursor := "  "
		if start+i == m.stackCursor {
			cursor = "➜ "
		}
		b.WriteString(fmt.Sprintf("%s%s / %s\n", cursor, stack.StackName, stack.ServiceName))
	}
	return b.String()
}

func (m *Model) visibleStackServices() []docker.StackService {
	if m.viewport.Height <= 0 || len(m.stackServices) == 0 {
		return nil
	}
	start := m.stackCursor - m.stackCursor%m.viewport.Height
	end := start + m.viewport.Height
	if end > len(m.stackServices) {
		end = len(m.stackServices)
	}
	return m.stackServices[start:end]
}
