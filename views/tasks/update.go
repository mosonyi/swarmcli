package tasksview

import (
	"swarmcli/ui"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case TasksLoadedMsg:
		if msg.Error != nil {
			l().Errorf("Error loading tasks: %v", msg.Error)
			return nil
		}
		m.tasks = msg.Tasks
		m.viewport.SetContent(m.renderTasks())
		return nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		// Calculate proper viewport dimensions accounting for frame overhead
		// Frame takes: top border (1), title line (1), header line (1), bottom border (1), footer (1) = 5 lines
		headerLines := 1
		footerLines := 1
		frameOverhead := 5

		contentHeight := msg.Height - frameOverhead - headerLines - footerLines
		if contentHeight < 5 {
			contentHeight = 5
		}

		contentWidth := msg.Width - ui.ComputeFrameDimensions(msg.Width, msg.Height, m.width, m.height, "", "").FrameWidth + msg.Width
		if contentWidth < 80 {
			contentWidth = 80
		}

		m.viewport.Width = contentWidth - 4
		m.viewport.Height = contentHeight
		return nil

	case tea.KeyMsg:
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return cmd
	}

	return nil
}
