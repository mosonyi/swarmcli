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
	m.nodeID = msg.NodeID
	m.entries = msg.Stacks
	m.cursor = 0

	if !m.ready {
		return
	}
	m.viewport.GotoTop()
	m.viewport.SetContent(m.buildContent())
	m.viewport.YOffset = 0
}

func (m *Model) buildContent() string {
	var b strings.Builder

	// Determine max stack name length
	maxLen := len("Stack Name") // at least the header length
	for _, stack := range m.entries {
		if l := len(stack.Name); l > maxLen {
			maxLen = l
		}
	}
	padding := maxLen + 2 // add some extra space

	// Header
	b.WriteString(fmt.Sprintf("%-*s %s\n", padding, "Stack Name", "Services"))
	b.WriteString(strings.Repeat("-", padding+8) + "\n") // underline

	// Stack rows
	for i, stack := range m.entries {
		cursor := "  "
		if i == m.cursor {
			cursor = "➜ "
		}
		b.WriteString(fmt.Sprintf("%s%-*s %d\n", cursor, padding, stack.Name, stack.ServiceCount))
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

	if m.cursor < m.viewport.YOffset {
		m.viewport.YOffset = m.cursor
	} else if m.cursor >= m.viewport.YOffset+h {
		m.viewport.YOffset = m.cursor - h + 1
	}
}
