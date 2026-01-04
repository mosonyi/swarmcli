package nodesview

import (
	"context"
	"fmt"
	"swarmcli/core/primitives/hash"
	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
	inspectview "swarmcli/views/inspect"
	servicesview "swarmcli/views/services"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case Msg:
		l().Infof("NodesView: Received Msg with %d entries", len(msg.Entries))
		// Update the hash with new data
		var err error
		newHash, err := hash.Compute(msg.Entries)
		if err != nil {
			l().Errorf("NodesView: Error computing hash: %v", err)
			return nil
		}

		// Only reset cursor on first load (when lastSnapshot is 0)
		shouldResetCursor := m.lastSnapshot == 0
		m.lastSnapshot = newHash

		// Store current cursor position before updating content
		oldCursor := m.List.Cursor
		oldYOffset := m.List.Viewport.YOffset

		m.SetContent(msg)

		// Restore cursor position unless it's the first load
		if !shouldResetCursor {
			// Make sure cursor is still valid after update
			if oldCursor < len(m.List.Filtered) {
				m.List.Cursor = oldCursor
				m.List.Viewport.YOffset = oldYOffset
			}
		}

		m.Visible = true
		return tickCmd()

	case TickMsg:
		l().Infof("NodesView: Received TickMsg, visible=%v", m.Visible)
		// Check for changes (this will return either a Msg or the next TickMsg)
		if m.Visible {
			return CheckNodesCmd(m.lastSnapshot)
		}
		// Continue polling even if not visible
		return tickCmd()

	case tea.WindowSizeMsg:
		m.List.Viewport.Width = msg.Width
		m.List.Viewport.Height = msg.Height
		m.ready = true
		// On first resize, reset YOffset to 0; on subsequent resizes, only reset if cursor is at top
		if m.firstResize {
			m.List.Viewport.YOffset = 0
			m.firstResize = false
		} else if m.List.Cursor == 0 {
			m.List.Viewport.YOffset = 0
		}
		return nil

	case tea.KeyMsg:
		// --- if in search mode, handle all keys via FilterableList ---
		if m.List.Mode == filterlist.ModeSearching {
			m.List.HandleKey(msg)
			return nil
		}

		// --- normal mode ---
		if msg.Type == tea.KeyEsc && m.List.Query != "" {
			m.List.Query = ""
			m.List.Mode = filterlist.ModeNormal
			m.List.ApplyFilter()
			m.List.Cursor = 0
			m.List.Viewport.GotoTop()
			return nil
		}

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

		return nil
	}

	var cmd tea.Cmd
	m.List.Viewport, cmd = m.List.Viewport.Update(msg)
	return cmd
}

func (m *Model) SetContent(msg Msg) {
	l().Infof("NodesView.SetContent: Updating display with %d entries", len(msg.Entries))

	m.List.Items = msg.Entries
	m.List.ApplyFilter()

	// Calculate column widths for all columns
	m.colWidths = calcColumnWidths(msg.Entries)
	m.setRenderItem()

	if m.ready {
		m.List.Viewport.SetContent(m.List.View())
		l().Info("NodesView.SetContent: Viewport content updated")
	} else {
		l().Warn("NodesView.SetContent: View not ready yet, skipping viewport update")
	}
}

func (m *Model) setRenderItem() {
	// Still need to call this for filterable list internals
	m.List.ComputeAndSetColWidth(func(n docker.NodeEntry) string {
		return n.Hostname
	}, 15)

	// Use bright white for content and reserve leading space in first column
	itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))

	m.List.RenderItem = func(n docker.NodeEntry, selected bool, colWidth int) string {
		// Compute proportional column widths for the current viewport width
		width := m.List.Viewport.Width
		if width <= 0 {
			width = 80
		}
		cols := 6
		starts := make([]int, cols)
		for i := 0; i < cols; i++ {
			starts[i] = (i * width) / cols
		}
		colWidths := make([]int, cols)
		for i := 0; i < cols; i++ {
			if i == cols-1 {
				colWidths[i] = width - starts[i]
			} else {
				colWidths[i] = starts[i+1] - starts[i]
			}
			if colWidths[i] < 1 {
				colWidths[i] = 1
			}
		}
		manager := "no"
		if n.Manager {
			manager = "yes"
		}
		labelsStr := formatLabels(n.Labels)
		// Use the pre-calculated column widths instead of the single colWidth
		if selected {
			selBg := lipgloss.Color("63")
			selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(selBg).Bold(true)
			// Preserve leading space for hostname when selected
			return selStyle.Render(fmt.Sprintf(" %-*s%-*s%-*s%-*s%-*s%-*s",
				colWidths[0]-1, n.Hostname,
				colWidths[1], n.Role,
				colWidths[2], n.State,
				colWidths[3], manager,
				colWidths[4], n.Addr,
				colWidths[5], labelsStr,
			))
		}

		// Ensure the first column has a leading space to align with header
		return itemStyle.Render(fmt.Sprintf(" %-*s%-*s%-*s%-*s%-*s%-*s",
			colWidths[0]-1, n.Hostname,
			colWidths[1], n.Role,
			colWidths[2], n.State,
			colWidths[3], manager,
			colWidths[4], n.Addr,
			colWidths[5], labelsStr,
		))
	}
}
