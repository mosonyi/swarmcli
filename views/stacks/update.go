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
	m.nodeStacks = msg.Stacks
	m.nodeServices = msg.Services
	m.nodeStackLines = strings.Split(msg.Output, "\n")
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
	b.WriteString("Stacks on node:\n\n")
	for i, stack := range m.nodeStacks {
		cursor := "  "
		if i == m.stackCursor {
			cursor = "➜ "
		}
		b.WriteString(fmt.Sprintf("%s%s\n", cursor, stack))
	}
	b.WriteString("\n[press enter to inspect logs, q/esc to go back]")
	return b.String()
}
