package stacksview

import (
	"fmt"
	"swarmcli/docker"
	"swarmcli/ui"
	servicesview "swarmcli/views/services"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

// Update handles all messages for the stacks view.
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {

	case Msg:
		m.nodeID = msg.NodeID
		m.setStacks(msg.Stacks)
		m.Visible = true
		return nil

	case RefreshErrorMsg:
		m.Visible = true
		m.List.Viewport.SetContent(fmt.Sprintf("Error refreshing stacks: %v", msg.Err))
		return nil

	case tea.WindowSizeMsg:
		m.List.Viewport.Width = msg.Width
		m.List.Viewport.Height = msg.Height
		m.ready = true
		m.List.Viewport.SetContent(m.List.View())
		return nil

	case tea.KeyMsg:
		m.List.HandleKey(msg)
		m.List.Viewport.SetContent(m.List.View())
		// Enter triggers navigation
		if msg.String() == "i" || msg.String() == "enter" {
			if m.List.Cursor < len(m.List.Filtered) {
				selected := m.List.Filtered[m.List.Cursor]
				return func() tea.Msg {
					return view.NavigateToMsg{
						ViewName: servicesview.ViewName,
						Payload:  map[string]interface{}{"stackName": selected.Name},
					}
				}
			}
		}
		return nil
	}

	var cmd tea.Cmd
	m.List.Viewport, cmd = m.List.Viewport.Update(msg)
	return cmd
}

func (m *Model) setStacks(stacks []docker.StackEntry) {
	m.List.Items = stacks
	m.List.Filtered = stacks
	m.List.Cursor = 0

	// Compute column width dynamically based on all stacks
	stackColWidth := m.List.ComputeColWidth(func(s docker.StackEntry) string {
		return s.Name
	}, 15)

	m.List.RenderItem = func(s docker.StackEntry, selected bool) string {
		line := fmt.Sprintf("%-*s %3d", stackColWidth, s.Name, s.ServiceCount)
		if selected {
			return ui.CursorStyle.Render(line)
		}
		return line
	}

	if m.ready {
		m.List.Viewport.SetContent(m.List.View())
	}
}
