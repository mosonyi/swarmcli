package helpview

import (
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Viewable.Width = msg.Width
		m.Viewable.Height = msg.Height
		m.width = msg.Width
		m.height = msg.Height
		return nil
	}

	var cmd tea.Cmd
	m.Viewable, cmd = m.Viewable.Update(msg)
	return cmd
}
