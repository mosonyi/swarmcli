// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package tasksview

import (
	"sort"
	"swarmcli/ui"
	helpview "swarmcli/views/help"
	view "swarmcli/views/view"

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
		// Reapply sorting to maintain sort order after data refresh
		m.applySorting()
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
		// Handle sorting keys
		switch msg.String() {
		case "N": // Shift+N: Sort by Name
			if m.sortField == SortByName {
				m.sortAscending = !m.sortAscending
			} else {
				m.sortField = SortByName
				m.sortAscending = true
			}
			m.applySorting()
			return nil
		case "S": // Shift+S: Sort by Service
			if m.sortField == SortByService {
				m.sortAscending = !m.sortAscending
			} else {
				m.sortField = SortByService
				m.sortAscending = true
			}
			m.applySorting()
			return nil
		case "D": // Shift+D: Sort by Node
			if m.sortField == SortByNode {
				m.sortAscending = !m.sortAscending
			} else {
				m.sortField = SortByNode
				m.sortAscending = true
			}
			m.applySorting()
			return nil
		case "T": // Shift+T: Sort by State
			if m.sortField == SortByState {
				m.sortAscending = !m.sortAscending
			} else {
				m.sortField = SortByState
				m.sortAscending = true
			}
			m.applySorting()
			return nil
		case "?":
			return func() tea.Msg {
				return view.NavigateToMsg{
					ViewName: view.NameHelp,
					Payload:  GetTasksHelpContent(),
				}
			}
		}

		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		return cmd
	}

	return nil
}

// applySorting applies the current sort configuration to the task list
func (m *Model) applySorting() {
	if len(m.tasks) == 0 {
		return
	}

	// Sort the task list
	switch m.sortField {
	case SortByName:
		sort.Slice(m.tasks, func(i, j int) bool {
			if m.sortAscending {
				return m.tasks[i].Name < m.tasks[j].Name
			}
			return m.tasks[i].Name > m.tasks[j].Name
		})
	case SortByService:
		sort.Slice(m.tasks, func(i, j int) bool {
			if m.sortAscending {
				return m.tasks[i].ServiceName < m.tasks[j].ServiceName
			}
			return m.tasks[i].ServiceName > m.tasks[j].ServiceName
		})
	case SortByNode:
		sort.Slice(m.tasks, func(i, j int) bool {
			if m.sortAscending {
				return m.tasks[i].NodeName < m.tasks[j].NodeName
			}
			return m.tasks[i].NodeName > m.tasks[j].NodeName
		})
	case SortByState:
		sort.Slice(m.tasks, func(i, j int) bool {
			if m.sortAscending {
				return m.tasks[i].DesiredState < m.tasks[j].DesiredState
			}
			return m.tasks[i].DesiredState > m.tasks[j].DesiredState
		})
	}

	// Re-render the viewport with sorted tasks
	m.viewport.SetContent(m.renderTasks())
	m.viewport.GotoTop()
}

// GetTasksHelpContent returns categorized help for the tasks view
func GetTasksHelpContent() []helpview.HelpCategory {
	return []helpview.HelpCategory{
		{
			Title: "View",
			Items: []helpview.HelpItem{
				{Keys: "<shift+n>", Description: "Order by Name"},
				{Keys: "<shift+s>", Description: "Order by Service"},
				{Keys: "<shift+d>", Description: "Order by Node"},
				{Keys: "<shift+t>", Description: "Order by State"},
			},
		},
		{
			Title: "Navigation",
			Items: []helpview.HelpItem{
				{Keys: "<↑/↓>", Description: "Scroll"},
				{Keys: "<pgup>", Description: "Page up"},
				{Keys: "<pgdown>", Description: "Page down"},
				{Keys: "<esc/q>", Description: "Back"},
			},
		},
	}
}
