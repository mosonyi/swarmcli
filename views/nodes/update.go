package nodesview

import (
	"context"
	"fmt"
	"swarmcli/docker"
	"swarmcli/ui"
	filterlist "swarmcli/ui/components/filterable/list"
	inspectview "swarmcli/views/inspect"
	servicesview "swarmcli/views/services"
	"swarmcli/views/view"

	"github.com/charmbracelet/lipgloss"
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
		// --- if in search mode, handle all keys via FilterableList ---
		if m.List.Mode == filterlist.ModeSearching {
			m.List.HandleKey(msg)
			return nil
		}

		// --- normal mode ---
		m.List.HandleKey(msg) // still handle up/down/pgup/pgdown

		// Enter triggers inspect / ps
		switch msg.String() {
		case "i":
			if m.List.Cursor < len(m.List.Filtered) {
				node := m.List.Filtered[m.List.Cursor]
				return func() tea.Msg {
					inspectContent, err := docker.Inspect(context.Background(), docker.InspectNode, node.ID)
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

	// Calculate column widths for all columns
	m.colWidths = calcColumnWidths(msg.Entries)
	m.setRenderItem()

	if m.ready {
		m.List.Viewport.SetContent(m.List.View())
		m.List.Viewport.GotoTop()
	}
}

func (m *Model) setRenderItem() {
	// Still need to call this for filterable list internals
	m.List.ComputeAndSetColWidth(func(n docker.NodeEntry) string {
		return n.Hostname
	}, 15)

	itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
	
	m.List.RenderItem = func(n docker.NodeEntry, selected bool, colWidth int) string {
		manager := "no"
		if n.Manager {
			manager = "yes"
		}
		// Use the pre-calculated column widths instead of the single colWidth
		line := fmt.Sprintf(
			"%-*s        %-*s        %-*s        %-*s        %-*s",
			m.colWidths["Hostname"], n.Hostname,
			m.colWidths["Role"], n.Role,
			m.colWidths["State"], n.State,
			m.colWidths["Manager"], manager,
			m.colWidths["Addr"], n.Addr,
		)
		if selected {
			return ui.CursorStyle.Render(line)
		}
		return itemStyle.Render(line)
	}
}
