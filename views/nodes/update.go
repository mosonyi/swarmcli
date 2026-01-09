package nodesview

import (
	"context"
	"fmt"
	"strings"
	"swarmcli/core/primitives/hash"
	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/confirmdialog"
	inspectview "swarmcli/views/inspect"
	servicesview "swarmcli/views/services"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// getFreshNodeState retrieves the current node state with a forced refresh
func getFreshNodeState(nodeID string) *docker.NodeEntry {
	// Force a synchronous refresh to get the absolute latest state
	snap, err := docker.RefreshSnapshot()
	if err != nil {
		l().Warnf("Failed to refresh snapshot: %v", err)
		snap = docker.GetSnapshot()
		if snap == nil {
			return nil
		}
	}
	entries := snap.ToNodeEntries()
	for _, entry := range entries {
		if entry.ID == nodeID {
			return &entry
		}
	}
	return nil
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case confirmdialog.ResultMsg:
		if !msg.Confirmed {
			// User cancelled, just close the dialog
			m.confirmDialog.Visible = false
			return nil
		}

		if m.List.Cursor < len(m.List.Filtered) {
			node := m.List.Filtered[m.List.Cursor]
			// Check which action to perform based on message content
			if strings.Contains(m.confirmDialog.Message, "Demote") {
				// Run demote, keeping dialog visible during operation
				return func() tea.Msg {
					if err := docker.DemoteNode(context.Background(), node.ID); err != nil {
						return DemoteErrorMsg{NodeID: node.ID, Error: err}
					}
					// Force refresh
					if _, err := docker.RefreshSnapshot(); err != nil {
						l().Warnf("Failed to refresh snapshot: %v", err)
					}
					// Return a message that will close dialog and refresh list
					return DemoteSuccessMsg{}
				}
			} else if strings.Contains(m.confirmDialog.Message, "Promote") {
				// Run promote, keeping dialog visible during operation
				return func() tea.Msg {
					if err := docker.PromoteNode(context.Background(), node.ID); err != nil {
						return PromoteErrorMsg{NodeID: node.ID, Error: err}
					}
					// Force refresh
					if _, err := docker.RefreshSnapshot(); err != nil {
						l().Warnf("Failed to refresh snapshot: %v", err)
					}
					// Return a message that will close dialog and refresh list
					return PromoteSuccessMsg{}
				}
			}
		}
		m.confirmDialog.Visible = false
		return nil

	case DemoteErrorMsg:
		// Reuse confirm dialog to display error
		m.confirmDialog.Visible = true
		m.confirmDialog.ErrorMode = true
		m.confirmDialog.Message = fmt.Sprintf("Failed to demote %s:\n%v", msg.NodeID, msg.Error)
		return nil

	case PromoteErrorMsg:
		// Reuse confirm dialog to display error
		m.confirmDialog.Visible = true
		m.confirmDialog.ErrorMode = true
		m.confirmDialog.Message = fmt.Sprintf("Failed to promote %s:\n%v", msg.NodeID, msg.Error)
		return nil

	case DemoteSuccessMsg:
		// Close dialog and reload nodes with fresh data
		m.confirmDialog.Visible = false
		return LoadNodesCmd()

	case PromoteSuccessMsg:
		// Close dialog and reload nodes with fresh data
		m.confirmDialog.Visible = false
		return LoadNodesCmd()

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
		if m.confirmDialog.Visible {
			return m.confirmDialog.Update(msg)
		}

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
		case "D":
			if m.List.Cursor < len(m.List.Filtered) {
				node := m.List.Filtered[m.List.Cursor]
				// Get fresh node state from snapshot to avoid stale data
				freshNode := getFreshNodeState(node.ID)
				if freshNode != nil {
					node = *freshNode
				}
				if node.Manager {
					m.confirmDialog.Visible = true
					m.confirmDialog.ErrorMode = false
					m.confirmDialog.Message = fmt.Sprintf("Demote node %q?", node.Hostname)
				} else {
					// Not a manager; show error dialog
					m.confirmDialog.Visible = true
					m.confirmDialog.ErrorMode = true
					m.confirmDialog.Message = fmt.Sprintf("Node %q is not a manager", node.Hostname)
				}
			}
		case "P":
			if m.List.Cursor < len(m.List.Filtered) {
				node := m.List.Filtered[m.List.Cursor]
				// Get fresh node state from snapshot to avoid stale data
				freshNode := getFreshNodeState(node.ID)
				if freshNode != nil {
					node = *freshNode
				}
				if !node.Manager {
					m.confirmDialog.Visible = true
					m.confirmDialog.ErrorMode = false
					m.confirmDialog.Message = fmt.Sprintf("Promote node %q?", node.Hostname)
				} else {
					// Already a manager; show error dialog
					m.confirmDialog.Visible = true
					m.confirmDialog.ErrorMode = true
					m.confirmDialog.Message = fmt.Sprintf("Node %q is already a manager", node.Hostname)
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
		return n.ID
	}, 15)

	// Use bright white for content and reserve leading space in first column
	itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))

	m.List.RenderItem = func(n docker.NodeEntry, selected bool, colWidth int) string {
		// Compute proportional column widths for the current viewport width
		width := m.List.Viewport.Width
		if width <= 0 {
			width = 80
		}
		cols := 8
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
		// Truncate ID so it fits the formatted field width (we render with
		// a leading space and field width `colWidths[0]-1`). Ensure the
		// final string length is <= colWidths[0]-1. If we need an ellipsis,
		// reserve 3 chars for it and trim the core accordingly.
		idStr := n.ID
		// safeWidth is the maximum length we can print for the ID (excluding the leading space)
		safeWidth := 0
		if colWidths[0] > 0 {
			safeWidth = colWidths[0] - 1
		}
		if safeWidth < 0 {
			safeWidth = 0
		}
		if len(idStr) > safeWidth {
			// If we can show at least 5 chars, show core + "..." and leave 1
			// extra char to avoid colliding with the next column: core + "..." so total == safeWidth
			if safeWidth > 4 {
				core := safeWidth - 4
				if core < 0 {
					core = 0
				}
				if core > len(idStr) {
					core = len(idStr)
				}
				idStr = idStr[:core] + "..."
			} else if safeWidth > 0 {
				// No room for ellipsis, just trim to fit
				if safeWidth > len(idStr) {
					// already fits, no-op
				} else {
					idStr = idStr[:safeWidth]
				}
			} else {
				idStr = ""
			}
		}
		// Use the pre-calculated column widths instead of the single colWidth
		if selected {
			selBg := lipgloss.Color("63")
			selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(selBg).Bold(true)
			// Preserve leading space for hostname when selected
			return selStyle.Render(fmt.Sprintf(" %-*s%-*s%-*s%-*s%-*s%-*s%-*s%-*s",
				colWidths[0]-1, idStr,
				colWidths[1], n.Hostname,
				colWidths[2], n.Role,
				colWidths[3], n.State,
				colWidths[4], manager,
				colWidths[5], n.Version,
				colWidths[6], n.Addr,
				colWidths[7], labelsStr,
			))
		}

		// Ensure the first column has a leading space to align with header
		return itemStyle.Render(fmt.Sprintf(" %-*s%-*s%-*s%-*s%-*s%-*s%-*s%-*s",
			colWidths[0]-1, idStr,
			colWidths[1], n.Hostname,
			colWidths[2], n.Role,
			colWidths[3], n.State,
			colWidths[4], manager,
			colWidths[5], n.Version,
			colWidths[6], n.Addr,
			colWidths[7], labelsStr,
		))
	}
}
