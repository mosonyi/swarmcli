package nodesview

import (
	"fmt"
	"swarmcli/docker"
	"swarmcli/ui"
	inspectview "swarmcli/views/inspect"
	servicesview "swarmcli/views/services"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case Msg:
		m.SetContent(msg)
		m.Visible = true
		return nil

	case tea.WindowSizeMsg:
		m.List.Viewport.Width = msg.Width
		m.List.Viewport.Height = msg.Height
		m.ready = true
		m.List.Viewport.SetContent(m.List.View())
		return nil

	case tea.KeyMsg:
		m.List.HandleKey(msg)

		// Enter triggers inspect / ps
		switch msg.String() {
		case "i":
			if m.List.Cursor < len(m.List.Filtered) {
				node := m.List.Filtered[m.List.Cursor]
				return func() tea.Msg {
					inspectContent, err := docker.Inspect(nil, docker.InspectNode, node.ID)
					if err != nil {
						inspectContent = "Error inspecting node: " + err.Error()
					}
					return view.NavigateToMsg{
						ViewName: inspectview.ViewName,
						Payload: map[string]interface{}{
							"title": "Node: " + node.Hostname,
							"json":  inspectContent,
						},
					}
				}
			}
		case "p":
			if m.List.Cursor < len(m.List.Filtered) {
				node := m.List.Filtered[m.List.Cursor]
				return func() tea.Msg {
					return view.NavigateToMsg{
						ViewName: servicesview.ViewName,
						Payload: map[string]interface{}{
							"nodeID":   node.ID,
							"hostname": node.Hostname,
						},
					}
				}
			}
		case "q":
			m.Visible = false
		}

		m.List.Viewport.SetContent(m.List.View())
		return nil
	}

	var cmd tea.Cmd
	m.List.Viewport, cmd = m.List.Viewport.Update(msg)
	return cmd
}

func (m *Model) SetContent(msg Msg) {
	m.List.Items = msg.Entries
	m.List.ApplyFilter()
	m.List.Cursor = 0

	m.setRenderItem()

	if m.ready {
		m.List.Viewport.SetContent(m.List.View())
		m.List.Viewport.GotoTop()
	}
}

func (m *Model) setRenderItem() {
	// Compute column widths based on all entries
	m.List.ComputeAndSetColWidth(func(n docker.NodeEntry) string {
		return n.Hostname
	}, 15)

	m.List.RenderItem = func(n docker.NodeEntry, selected bool, colWidth int) string {
		manager := "no"
		if n.Manager {
			manager = "yes"
		}
		line := fmt.Sprintf(
			"%-*s  %-*s  %-*s  %-*s  %-*s",
			colWidth, n.Hostname,
			colWidth, n.Role,
			colWidth, n.State,
			colWidth, manager,
			colWidth, n.Addr,
		)
		if selected {
			return ui.CursorStyle.Render(line)
		}
		return line
	}
}
