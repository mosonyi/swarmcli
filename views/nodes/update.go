package nodesview

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"swarmcli/core/primitives/hash"
	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/confirmdialog"
	helpview "swarmcli/views/help"
	inspectview "swarmcli/views/inspect"
	servicesview "swarmcli/views/services"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types/swarm"
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
			} else if strings.Contains(m.confirmDialog.Message, "Remove") {
				// Run remove with force=true, keeping dialog visible during operation
				return func() tea.Msg {
					if err := docker.RemoveNode(context.Background(), node.ID, true); err != nil {
						return RemoveErrorMsg{NodeID: node.ID, Error: err}
					}
					// Force refresh
					if _, err := docker.RefreshSnapshot(); err != nil {
						l().Warnf("Failed to refresh snapshot: %v", err)
					}
					// Return a message that will close dialog and refresh list
					return RemoveSuccessMsg{}
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

	case RemoveErrorMsg:
		// Reuse confirm dialog to display error
		m.confirmDialog.Visible = true
		m.confirmDialog.ErrorMode = true
		m.confirmDialog.Message = fmt.Sprintf("Failed to remove %s:\n%v", msg.NodeID, msg.Error)
		return nil

	case RemoveSuccessMsg:
		// Close dialog and reload nodes with fresh data
		m.confirmDialog.Visible = false
		return LoadNodesCmd()

	case SetAvailabilityErrorMsg:
		// Show error in confirm dialog
		m.confirmDialog.Visible = true
		m.confirmDialog.ErrorMode = true
		m.confirmDialog.Message = fmt.Sprintf("Failed to set availability:\n%v", msg.Error)
		return nil

	case SetAvailabilitySuccessMsg:
		// Close dialog and reload nodes
		m.availabilityDialog = false
		return LoadNodesCmd()

	case AddLabelErrorMsg:
		// Show error in confirm dialog
		m.confirmDialog.Visible = true
		m.confirmDialog.ErrorMode = true
		m.confirmDialog.Message = fmt.Sprintf("Failed to add label:\n%v", msg.Error)
		return nil

	case AddLabelSuccessMsg:
		// Close dialog and reload nodes
		m.labelInputDialog = false
		return LoadNodesCmd()

	case RemoveLabelErrorMsg:
		// Show error in confirm dialog
		m.confirmDialog.Visible = true
		m.confirmDialog.ErrorMode = true
		m.confirmDialog.Message = fmt.Sprintf("Failed to remove label:\n%v", msg.Error)
		return nil

	case RemoveLabelSuccessMsg:
		// Close dialog and reload nodes
		m.labelRemoveDialog = false
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
		if m.labelInputDialog {
			return m.handleLabelInputDialogKey(msg)
		}

		if m.labelRemoveDialog {
			return m.handleLabelRemoveDialogKey(msg)
		}

		if m.availabilityDialog {
			return m.handleAvailabilityDialogKey(msg)
		}

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

		// Handle left/right for labels scrolling
		switch msg.String() {
		case "left":
			if m.labelsScrollOffset > 0 {
				m.labelsScrollOffset -= 5
				if m.labelsScrollOffset < 0 {
					m.labelsScrollOffset = 0
				}
				m.setRenderItem() // Re-render with new offset
				m.List.Viewport.SetContent(m.List.View())
			}
			return nil
		case "right":
			if m.List.Cursor < len(m.List.Filtered) {
				node := m.List.Filtered[m.List.Cursor]
				labelsStr := formatLabels(node.Labels)
				// Allow scrolling if labels are longer than visible width
				if len(labelsStr) > m.labelsScrollOffset+20 {
					m.labelsScrollOffset += 5
					m.setRenderItem() // Re-render with new offset
					m.List.Viewport.SetContent(m.List.View())
				}
			}
			return nil
		}

		oldCursor := m.List.Cursor
		m.List.HandleKey(msg) // still handle up/down/pgup/pgdown

		// Reset scroll offset on cursor movement
		if m.List.Cursor != oldCursor {
			m.labelsScrollOffset = 0
			m.setRenderItem()
			m.List.Viewport.SetContent(m.List.View())
		}

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
		case "?":
			return func() tea.Msg {
				return view.NavigateToMsg{
					ViewName: "help",
					Payload:  GetNodesHelpContent(),
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
		case "a":
			if m.List.Cursor < len(m.List.Filtered) {
				node := m.List.Filtered[m.List.Cursor]
				m.availabilityDialog = true
				m.availabilityNodeID = node.ID
				m.availabilitySelection = 0
			}
		case "ctrl+l":
			if m.List.Cursor < len(m.List.Filtered) {
				node := m.List.Filtered[m.List.Cursor]
				m.labelInputDialog = true
				m.labelInputNodeID = node.ID
				m.labelInputValue = ""
			}
		case "ctrl+r":
			if m.List.Cursor < len(m.List.Filtered) {
				node := m.List.Filtered[m.List.Cursor]
				if len(node.Labels) == 0 {
					m.confirmDialog.Visible = true
					m.confirmDialog.ErrorMode = true
					m.confirmDialog.Message = "Node has no labels to remove"
				} else {
					// Build label list as "key=value" strings
					labels := make([]string, 0, len(node.Labels))
					for k, v := range node.Labels {
						labels = append(labels, k+"="+v)
					}
					// Sort for consistent display
					sort.Strings(labels)
					m.labelRemoveDialog = true
					m.labelRemoveNodeID = node.ID
					m.labelRemoveSelection = 0
					m.labelRemoveLabels = labels
				}
			}
		case "ctrl+d":
			if m.List.Cursor < len(m.List.Filtered) {
				node := m.List.Filtered[m.List.Cursor]
				m.confirmDialog.Visible = true
				m.confirmDialog.ErrorMode = false
				m.confirmDialog.Message = fmt.Sprintf("Remove node %q from swarm?\nWarning: This action cannot be undone.", node.Hostname)
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
		cols := 9
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
		labelsStr := formatLabelsWithScroll(n.Labels, m.labelsScrollOffset, colWidths[7])
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
			return selStyle.Render(fmt.Sprintf(" %-*s%-*s%-*s%-*s%-*s%-*s%-*s%-*s%-*s",
				colWidths[0]-1, idStr,
				colWidths[1], n.Hostname,
				colWidths[2], n.Role,
				colWidths[3], n.State,
				colWidths[4], n.Availability,
				colWidths[5], manager,
				colWidths[6], n.Version,
				colWidths[7], n.Addr,
				colWidths[8], labelsStr,
			))
		}

		// Ensure the first column has a leading space to align with header
		return itemStyle.Render(fmt.Sprintf(" %-*s%-*s%-*s%-*s%-*s%-*s%-*s%-*s%-*s",
			colWidths[0]-1, idStr,
			colWidths[1], n.Hostname,
			colWidths[2], n.Role,
			colWidths[3], n.State,
			colWidths[4], n.Availability,
			colWidths[5], manager,
			colWidths[6], n.Version,
			colWidths[7], n.Addr,
			colWidths[8], labelsStr,
		))
	}
}

// handleAvailabilityDialogKey handles key presses when availability dialog is visible
func (m *Model) handleAvailabilityDialogKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.availabilitySelection > 0 {
			m.availabilitySelection--
		}
	case "down", "j":
		if m.availabilitySelection < 2 {
			m.availabilitySelection++
		}
	case "1", "a":
		m.availabilitySelection = 0
	case "2", "p":
		m.availabilitySelection = 1
	case "3", "d":
		m.availabilitySelection = 2
	case "enter":
		// Apply the selected availability
		availability := []string{"active", "pause", "drain"}[m.availabilitySelection]
		nodeID := m.availabilityNodeID
		m.availabilityDialog = false
		return func() tea.Msg {
			var avail swarm.NodeAvailability
			switch availability {
			case "active":
				avail = swarm.NodeAvailabilityActive
			case "pause":
				avail = swarm.NodeAvailabilityPause
			case "drain":
				avail = swarm.NodeAvailabilityDrain
			}
			if err := docker.SetNodeAvailability(context.Background(), nodeID, avail); err != nil {
				return SetAvailabilityErrorMsg{NodeID: nodeID, Error: err}
			}
			// Force refresh
			if _, err := docker.RefreshSnapshot(); err != nil {
				l().Warnf("Failed to refresh snapshot: %v", err)
			}
			return SetAvailabilitySuccessMsg{}
		}
	case "esc", "q":
		m.availabilityDialog = false
	}
	return nil
}

// handleLabelInputDialogKey handles key presses when label input dialog is visible
func (m *Model) handleLabelInputDialogKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.Type {
	case tea.KeyEsc:
		m.labelInputDialog = false
		m.labelInputValue = ""
		return nil
	case tea.KeyEnter:
		// Parse and apply the label
		input := strings.TrimSpace(m.labelInputValue)
		if input == "" {
			m.labelInputDialog = false
			return nil
		}

		// Parse key=value format
		parts := strings.SplitN(input, "=", 2)
		if len(parts) != 2 {
			m.confirmDialog.Visible = true
			m.confirmDialog.ErrorMode = true
			m.confirmDialog.Message = "Invalid format. Use: key=value"
			m.labelInputDialog = false
			return nil
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		if key == "" {
			m.confirmDialog.Visible = true
			m.confirmDialog.ErrorMode = true
			m.confirmDialog.Message = "Label key cannot be empty"
			m.labelInputDialog = false
			return nil
		}

		nodeID := m.labelInputNodeID
		m.labelInputDialog = false
		m.labelInputValue = ""

		return func() tea.Msg {
			if err := docker.AddNodeLabel(context.Background(), nodeID, key, value); err != nil {
				return AddLabelErrorMsg{NodeID: nodeID, Error: err}
			}
			// Force refresh
			if _, err := docker.RefreshSnapshot(); err != nil {
				l().Warnf("Failed to refresh snapshot: %v", err)
			}
			return AddLabelSuccessMsg{}
		}
	case tea.KeyBackspace:
		if len(m.labelInputValue) > 0 {
			m.labelInputValue = m.labelInputValue[:len(m.labelInputValue)-1]
		}
	case tea.KeyRunes:
		m.labelInputValue += string(msg.Runes)
	}
	return nil
}

// handleLabelRemoveDialogKey handles key presses when label remove dialog is visible
func (m *Model) handleLabelRemoveDialogKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "up", "k":
		if m.labelRemoveSelection > 0 {
			m.labelRemoveSelection--
		}
	case "down", "j":
		if m.labelRemoveSelection < len(m.labelRemoveLabels)-1 {
			m.labelRemoveSelection++
		}
	case "enter":
		// Parse selected label and remove it
		if m.labelRemoveSelection < len(m.labelRemoveLabels) {
			selected := m.labelRemoveLabels[m.labelRemoveSelection]
			// Extract key from "key=value"
			parts := strings.SplitN(selected, "=", 2)
			if len(parts) < 1 {
				m.labelRemoveDialog = false
				return nil
			}
			key := parts[0]
			nodeID := m.labelRemoveNodeID
			m.labelRemoveDialog = false

			return func() tea.Msg {
				if err := docker.RemoveNodeLabel(context.Background(), nodeID, key); err != nil {
					return RemoveLabelErrorMsg{NodeID: nodeID, Error: err}
				}
				// Force refresh
				if _, err := docker.RefreshSnapshot(); err != nil {
					l().Warnf("Failed to refresh snapshot: %v", err)
				}
				return RemoveLabelSuccessMsg{}
			}
		}
	case "esc", "q":
		m.labelRemoveDialog = false
	}
	return nil
}

// GetNodesHelpContent returns categorized help for the nodes view
func GetNodesHelpContent() []helpview.HelpCategory {
	return []helpview.HelpCategory{
		{
			Title: "General",
			Items: []helpview.HelpItem{
				{Keys: "<i>", Description: "Inspect node"},
				{Keys: "<p>", Description: "Show services on node"},
				{Keys: "<a>", Description: "Change availability"},
				{Keys: "<ctrl+l>", Description: "Add label to node"},
				{Keys: "<ctrl+r>", Description: "Remove label from node"},
				{Keys: "<shift+p>", Description: "Promote to manager"},
				{Keys: "<shift+d>", Description: "Demote to worker"},
				{Keys: "<ctrl+d>", Description: "Remove node"},
				{Keys: "</>", Description: "Filter"},
			},
		},
		{
			Title: "View",
			Items: []helpview.HelpItem{
				{Keys: "<shift+h>", Description: "Order by Hostname (todo)"},
				{Keys: "<shift+s>", Description: "Order by Status (todo)"},
				{Keys: "<shift+a>", Description: "Order by Availability (todo)"},
				{Keys: "<shift+r>", Description: "Order by Role (todo)"},
			},
		},
		{
			Title: "Navigation",
			Items: []helpview.HelpItem{
				{Keys: "<↑/↓>", Description: "Navigate"},
				{Keys: "<pgup>", Description: "Page up"},
				{Keys: "<pgdown>", Description: "Page down"},
				{Keys: "<q>", Description: "Back to stacks"},
			},
		},
	}
}
