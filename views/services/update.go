// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package servicesview

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
	logsview "swarmcli/views/logs"
	"swarmcli/views/scaledialog"
	"swarmcli/views/view"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/docker/docker/api/types/swarm"
)

// truncateWithEllipsis truncates a string to maxWidth, adding … if needed
func truncateWithEllipsis(s string, maxWidth int) string {
	if len(s) <= maxWidth {
		return s
	}
	if maxWidth <= 1 {
		return "…"
	}
	if maxWidth == 2 {
		return s[:1] + "…"
	}
	return s[:maxWidth-1] + "…"
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {

	case Msg:
		l().Infof("ServicesView: Received Msg with %d entries", len(msg.Entries))

		// If we're viewing a specific stack or node and all services are gone, show message
		// This handles cases where:
		// - All services in a stack are deleted (stack no longer exists)
		// - All services on a node are removed
		if len(msg.Entries) == 0 && (msg.FilterType == StackFilter || msg.FilterType == NodeFilter) {
			l().Info("ServicesView: No services remaining in filtered view, showing dialog")
			m.pendingAction = "empty-stack"
			m.confirmDialog.Visible = true
			m.confirmDialog.ErrorMode = true
			var target string
			if msg.FilterType == StackFilter {
				target = "Stack"
			} else {
				target = "Node"
			}
			m.confirmDialog.Message = fmt.Sprintf("%s no longer has services, returning to stacks view.\n\nPress Enter to continue.", target)
			// Don't update content or set visible - just show the dialog
			return nil
		}

		// Update the hash with new data
		var err error
		m.lastSnapshot, err = hash.Compute(msg.Entries)
		if err != nil {
			l().Errorf("ServicesView: Error computing hash: %v", err)
			return nil
		}
		m.SetContent(msg)
		m.Visible = true
		// Continue polling and refresh tasks for any expanded services
		return tea.Batch(tickCmd(), m.refreshExpandedTasksCmd(m.expandedServices))

	case TickMsg:
		l().Infof("ServicesView: Received TickMsg, visible=%v", m.Visible)
		// Check for changes and refresh expanded tasks
		if m.Visible {
			m.refreshServiceErrorsFromSnapshot()
			if m.ready {
				m.List.Viewport.SetContent(m.List.View())
			}
			return tea.Batch(
				m.checkServicesCmd(m.lastSnapshot, m.filterType, m.nodeID, m.stackName),
				m.refreshExpandedTasksCmd(m.expandedServices),
			)
		}
		// Continue polling even if not visible
		return tickCmd()

	case TasksLoadedMsg:
		// Store loaded tasks - view will automatically re-render
		m.serviceTasks[msg.ServiceID] = msg.Tasks
		m.setRenderItem()
		return nil

	case tea.WindowSizeMsg:
		m.List.Viewport.Width = msg.Width
		m.List.Viewport.Height = msg.Height
		m.List.SetOuterSize(msg.Width, msg.Height)
		m.ready = true
		// On first resize, reset YOffset to 0; on subsequent resizes, only reset if cursor is at top
		if m.firstResize {
			m.List.Viewport.YOffset = 0
			m.firstResize = false
		} else if m.List.Cursor == 0 {
			m.List.Viewport.YOffset = 0
		}
		return nil
	case scaledialog.ResultMsg:
		m.scaleDialog.Visible = false
		if msg.Confirmed && m.List.Cursor < len(m.List.Filtered) {
			entry := m.List.Filtered[m.List.Cursor]
			l().Infof("Scaling service %s to %d replicas", entry.ServiceName, msg.Replicas)
			serviceOps := m.deps.Services
			snapshotOps := m.deps.Snapshot
			refreshCmd := m.refreshServicesCmd(m.nodeID, m.stackName, m.filterType)
			return func() tea.Msg {
				if err := serviceOps.ScaleService(entry.ServiceID, msg.Replicas); err != nil {
					l().Errorf("Failed to scale service %s: %v", entry.ServiceName, err)
					return ScaleErrorMsg{
						ServiceName: entry.ServiceName,
						Error:       err,
					}
				}
				l().Infof("Successfully scaled service %s to %d replicas", entry.ServiceName, msg.Replicas)
				// Force immediate snapshot refresh
				if _, err := snapshotOps.RefreshSnapshot(); err != nil {
					l().Warnf("Failed to refresh snapshot: %v", err)
				}
				return refreshCmd()
			}
		}
		return nil

	case confirmdialog.ResultMsg:
		m.confirmDialog.Visible = false

		if msg.Confirmed && m.List.Cursor < len(m.List.Filtered) {
			entry := m.List.Filtered[m.List.Cursor]
			serviceOps := m.deps.Services
			snapshotOps := m.deps.Snapshot
			refreshCmd := m.refreshServicesCmd(m.nodeID, m.stackName, m.filterType)

			switch m.pendingAction {
			case "remove":
				l().Debugln("Starting remove for", entry.ServiceName)
				return func() tea.Msg {
					l().Infof("Executing remove for service: %s", entry.ServiceName)
					if err := serviceOps.RemoveService(entry.ServiceName); err != nil {
						l().Errorf("Failed to remove service %s: %v", entry.ServiceName, err)
						return RemoveErrorMsg{
							ServiceName: entry.ServiceName,
							Error:       err,
						}
					}
					l().Infof("Successfully removed service: %s", entry.ServiceName)
					// Force immediate snapshot refresh
					if _, err := snapshotOps.RefreshSnapshot(); err != nil {
						l().Warnf("Failed to refresh snapshot: %v", err)
					}
					return refreshCmd()
				}
			case "rollback":
				l().Debugln("Starting rollback for", entry.ServiceName)
				return func() tea.Msg {
					l().Infof("Executing rollback for service: %s", entry.ServiceName)
					if err := serviceOps.RollbackService(entry.ServiceName); err != nil {
						l().Errorf("Failed to rollback service %s: %v", entry.ServiceName, err)
						return RollbackErrorMsg{
							ServiceName: entry.ServiceName,
							Error:       err,
						}
					}
					l().Infof("Successfully rolled back service: %s", entry.ServiceName)
					// Force immediate snapshot refresh
					if _, err := snapshotOps.RefreshSnapshot(); err != nil {
						l().Warnf("Failed to refresh snapshot: %v", err)
					}
					return refreshCmd()
				}
			default:
				// Default to restart
				l().Debugln("Starting restart for", entry.ServiceName)
				return func() tea.Msg {
					l().Infof("Executing restart for service: %s", entry.ServiceName)
					if err := serviceOps.RestartService(entry.ServiceName); err != nil {
						l().Errorf("Failed to restart service %s: %v", entry.ServiceName, err)
						return RestartErrorMsg{
							ServiceName: entry.ServiceName,
							Error:       err,
						}
					}
					l().Infof("Successfully restarted service: %s", entry.ServiceName)
					// Force immediate snapshot refresh
					if _, err := snapshotOps.RefreshSnapshot(); err != nil {
						l().Warnf("Failed to refresh snapshot: %v", err)
					}
					return refreshCmd()
				}
			}
		}

		// Handle empty-stack navigation
		if m.pendingAction == "empty-stack" {
			m.pendingAction = ""
			m.Visible = false
			return func() tea.Msg {
				return view.NavigateToMsg{ViewName: view.NameStacks, Payload: nil, Replace: true}
			}
		}

		m.pendingAction = ""
		return nil

	case RestartErrorMsg:
		// Show error in a confirm dialog (reusing it as an error display)
		m.confirmDialog.Visible = true
		m.confirmDialog.ErrorMode = true
		m.confirmDialog.Message = fmt.Sprintf("Failed to restart %s:\n%v", msg.ServiceName, msg.Error)
		return nil

	case ScaleErrorMsg:
		// Show error in a confirm dialog (reusing it as an error display)
		m.confirmDialog.Visible = true
		m.confirmDialog.ErrorMode = true
		m.confirmDialog.Message = fmt.Sprintf("Failed to scale %s:\n%v", msg.ServiceName, msg.Error)
		return nil

	case RemoveErrorMsg:
		// Show error in a confirm dialog (reusing it as an error display)
		m.confirmDialog.Visible = true
		m.confirmDialog.ErrorMode = true
		m.confirmDialog.Message = fmt.Sprintf("Failed to remove %s:\n%v", msg.ServiceName, msg.Error)
		return nil

	case RollbackErrorMsg:
		// Show error in a confirm dialog (reusing it as an error display)
		m.confirmDialog.Visible = true
		m.confirmDialog.ErrorMode = true
		m.confirmDialog.Message = fmt.Sprintf("Failed to rollback %s:\n%v", msg.ServiceName, msg.Error)
		return nil

	case tea.KeyMsg:
		// If this view is not visible, don't process keys
		// This allows overlaid views (like logs) to handle input
		if !m.Visible {
			return nil
		}

		if m.confirmDialog.Visible {
			return m.confirmDialog.Update(msg)
		}

		if m.scaleDialog.Visible {
			return m.scaleDialog.Update(msg)
		}

		// --- if in search mode, handle all keys via FilterableList ---
		if m.List.Mode == filterlist.ModeSearching {
			m.List.HandleKey(msg)
			return nil
		}

		// Handle left/right for column scrolling
		switch msg.String() {
		case "left":
			if m.columnScrollOffset > 0 {
				m.columnScrollOffset -= 5
				if m.columnScrollOffset < 0 {
					m.columnScrollOffset = 0
				}
				m.setRenderItem()
				m.List.Viewport.SetContent(m.List.View())
			}
			return nil
		case "right":
			// Scroll right if any column has more content
			m.columnScrollOffset += 5
			m.setRenderItem()
			m.List.Viewport.SetContent(m.List.View())
			return nil
		}

		// --- normal mode ---
		if msg.Type == tea.KeyEsc && m.List.Query != "" {
			m.List.Query = ""
			m.List.Mode = filterlist.ModeNormal
			m.List.ApplyFilter()
			m.List.Cursor = 0
			m.List.Viewport.GotoTop()
			m.selectedTaskIndex = -1
			return nil
		}

		// Handle task navigation for expanded services
		if m.List.Cursor < len(m.List.Filtered) {
			entry := m.List.Filtered[m.List.Cursor]
			if m.expandedServices[entry.ServiceID] {
				tasks := m.serviceTasks[entry.ServiceID]
				switch msg.String() {
				case "down":
					if m.selectedTaskIndex < len(tasks)-1 {
						// Move down within tasks or from service to first task
						m.selectedTaskIndex++
						m.setRenderItem()
						return nil
					} else if m.selectedTaskIndex == len(tasks)-1 {
						// At last task, move to next service
						m.selectedTaskIndex = -1
						m.List.HandleKey(msg)
						return nil
					}
					// If selectedTaskIndex == -1, fall through to normal handling
				case "up":
					if m.selectedTaskIndex > 0 {
						// Move up within tasks
						m.selectedTaskIndex--
						m.setRenderItem()
						return nil
					} else if m.selectedTaskIndex == 0 {
						// At first task, move back to service row
						m.selectedTaskIndex = -1
						m.setRenderItem()
						return nil
					}
					// If selectedTaskIndex == -1, fall through to normal handling
				}
			} else if msg.String() == "up" && m.selectedTaskIndex == -1 && m.List.Cursor > 0 {
				// At a service row, check if previous service has expanded tasks
				prevEntry := m.List.Filtered[m.List.Cursor-1]
				if m.expandedServices[prevEntry.ServiceID] {
					prevTasks := m.serviceTasks[prevEntry.ServiceID]
					if len(prevTasks) > 0 {
						// Move to last task of previous service
						m.List.Cursor--
						m.selectedTaskIndex = len(prevTasks) - 1
						m.setRenderItem()
						return nil
					}
				}
			}
		}

		// Store old cursor to detect changes
		oldCursor := m.List.Cursor
		m.List.HandleKey(msg) // handle up/down/pgup/pgdown

		// Reset task selection when moving to different service
		if oldCursor != m.List.Cursor {
			m.selectedTaskIndex = -1
			// Reset horizontal scroll when moving cursor
			m.columnScrollOffset = 0
		}

		switch msg.String() {
		case "s":
			if m.List.Cursor < len(m.List.Filtered) {
				entry := m.List.Filtered[m.List.Cursor]
				m.scaleDialog.Show(entry.ServiceName, uint64(entry.ReplicasTotal))
			}
		case "p":
			// Toggle tasks expansion for selected service
			if m.List.Cursor < len(m.List.Filtered) {
				entry := m.List.Filtered[m.List.Cursor]
				// Toggle expansion state
				m.expandedServices[entry.ServiceID] = !m.expandedServices[entry.ServiceID]

				// If expanding, fetch tasks
				if m.expandedServices[entry.ServiceID] {
					taskOps := m.deps.Tasks
					return func() tea.Msg {
						tasks, err := taskOps.GetTasksForService(entry.ServiceID)
						if err != nil {
							l().Errorf("Failed to fetch tasks for service %s: %v", entry.ServiceName, err)
							// Still toggle to show empty state
							tasks = []docker.TaskEntry{}
						}
						return TasksLoadedMsg{
							ServiceID: entry.ServiceID,
							Tasks:     tasks,
						}
					}
				} else {
					// Collapsing - remove cached tasks and let view re-render
					delete(m.serviceTasks, entry.ServiceID)
					m.selectedTaskIndex = -1
					m.setRenderItem()
				}
			}
		case "i":
			if m.List.Cursor < len(m.List.Filtered) {
				entry := m.List.Filtered[m.List.Cursor]
				inspectOps := m.deps.Inspect
				return func() tea.Msg {
					content, err := inspectOps.Inspect(context.Background(), docker.InspectService, entry.ServiceID)
					if err != nil {
						content = fmt.Sprintf("Error inspecting service %q: %v", entry.ServiceName, err)
					}
					return view.NavigateToMsg{
						ViewName: inspectview.ViewName,
						Payload: map[string]interface{}{
							"title": fmt.Sprintf("Service: %s", entry.ServiceName),
							"json":  content,
						},
					}
				}
			}
		case "r":
			if m.List.Cursor < len(m.List.Filtered) {
				entry := m.List.Filtered[m.List.Cursor]
				m.pendingAction = "restart"
				m.confirmDialog.Visible = true
				m.confirmDialog.ErrorMode = false
				m.confirmDialog.Message = fmt.Sprintf("Restart service %q?", entry.ServiceName)
			}
		case "?":
			return func() tea.Msg {
				return view.NavigateToMsg{
					ViewName: view.NameHelp,
					Payload:  GetServicesHelpContent(),
				}
			}
		case "ctrl+d":
			if m.List.Cursor < len(m.List.Filtered) {
				entry := m.List.Filtered[m.List.Cursor]
				m.pendingAction = "remove"
				m.confirmDialog.Visible = true
				m.confirmDialog.ErrorMode = false
				m.confirmDialog.Message = fmt.Sprintf("Remove service %q?\n\nThis action cannot be undone!", entry.ServiceName)
			}
		case "ctrl+r":
			if m.List.Cursor < len(m.List.Filtered) {
				entry := m.List.Filtered[m.List.Cursor]
				m.pendingAction = "rollback"
				m.confirmDialog.Visible = true
				m.confirmDialog.ErrorMode = false
				m.confirmDialog.Message = fmt.Sprintf("Rollback service %q to previous configuration?", entry.ServiceName)
			}
		case "l":
			if m.List.Cursor < len(m.List.Filtered) {
				entry := m.List.Filtered[m.List.Cursor]
				return func() tea.Msg {
					return view.NavigateToMsg{
						Payload:  entry,
						ViewName: logsview.ViewName,
					}
				}
			}
		case "x":
			if m.List.Cursor >= len(m.List.Filtered) {
				return nil
			}
			entry := m.List.Filtered[m.List.Cursor]
			action, ok := view.GetAction("shell")
			if !ok {
				m.confirmDialog.Visible = true
				m.confirmDialog.ErrorMode = true
				m.confirmDialog.Message = view.BEUnavailableErr("Shell").Error()
				return nil
			}
			return action(entry.ServiceName)
		case "w":
			if m.List.Cursor >= len(m.List.Filtered) {
				return nil
			}
			entry := m.List.Filtered[m.List.Cursor]
			action, ok := view.GetAction("port-forward")
			if !ok {
				m.confirmDialog.Visible = true
				m.confirmDialog.ErrorMode = true
				m.confirmDialog.Message = view.BEUnavailableErr("Port Forward").Error()
				return nil
			}
			return action(entry.ServiceName)
		case "q":
			m.Visible = false
			// Go back to stacks view
			return func() tea.Msg { return view.NavigateToMsg{ViewName: view.NameStacks, Payload: nil} }

		case "esc":
			// ESC should also go back to stacks view
			m.Visible = false
			return func() tea.Msg { return view.NavigateToMsg{ViewName: view.NameStacks, Payload: nil} }

		// Sort by Name (Shift+N)
		case "N":
			if m.sortField == SortByName {
				m.sortAscending = !m.sortAscending
			} else {
				m.sortField = SortByName
				m.sortAscending = true
			}
			m.applySorting()
			return nil

		// Sort by Status (Shift+S)
		case "S":
			if m.sortField == SortByStatus {
				m.sortAscending = !m.sortAscending
			} else {
				m.sortField = SortByStatus
				m.sortAscending = true
			}
			m.applySorting()
			return nil

		// Sort by Image (Shift+I)
		case "I":
			if m.sortField == SortByImage {
				m.sortAscending = !m.sortAscending
			} else {
				m.sortField = SortByImage
				m.sortAscending = true
			}
			m.applySorting()
			return nil

		// Sort by Ports (Shift+P)
		case "P":
			if m.sortField == SortByPorts {
				m.sortAscending = !m.sortAscending
			} else {
				m.sortField = SortByPorts
				m.sortAscending = true
			}
			m.applySorting()
			return nil

		// Sort by Created (Shift+C)
		case "C":
			if m.sortField == SortByCreated {
				m.sortAscending = !m.sortAscending
			} else {
				m.sortField = SortByCreated
				m.sortAscending = true
			}
			m.applySorting()
			return nil

		// Sort by Updated (Shift+U)
		case "U":
			if m.sortField == SortByUpdated {
				m.sortAscending = !m.sortAscending
			} else {
				m.sortField = SortByUpdated
				m.sortAscending = true
			}
			m.applySorting()
			return nil

		// Sort by Error (Shift+R)
		case "R":
			if m.sortField == SortByError {
				m.sortAscending = !m.sortAscending
			} else {
				m.sortField = SortByError
				m.sortAscending = true
			}
			m.applySorting()
			return nil
		}

		m.List.Viewport.SetContent(m.List.View())
		return nil
	}

	var cmd tea.Cmd
	m.List.Viewport, cmd = m.List.Viewport.Update(msg)
	return cmd
}

// formatErrorWithScroll formats error text with horizontal offset and truncation indicator
func formatErrorWithScroll(full string, offset int, maxWidth int) string {
	if full == "" {
		return ""
	}
	if offset > len(full) {
		offset = len(full)
	}
	visible := full[offset:]

	// Truncate to maxWidth
	if len(visible) > maxWidth {
		visible = truncateWithEllipsis(visible, maxWidth)
	}
	return visible
}

func (m *Model) SetContent(msg Msg) {
	l().Infof("ServicesView.SetContent: Updating display with %d services", len(msg.Entries))

	m.title = msg.Title

	m.List.Items = msg.Entries
	m.List.ApplyFilter()

	// Compute per-service error summary from cached snapshot so errors appear
	// without expanding tasks.
	m.refreshServiceErrorsFromSnapshot()

	// Reapply sorting to maintain sort order after data refresh
	m.applySorting()

	// If we were navigated here with a target service to select, select it now
	// (after filtering/sorting so indices match what the user sees).
	if m.pendingSelectServiceName != "" {
		for i := range m.List.Filtered {
			if m.List.Filtered[i].ServiceName == m.pendingSelectServiceName {
				m.List.Cursor = i
				break
			}
		}
		m.pendingSelectServiceName = ""
	}

	m.filterType = msg.FilterType
	m.nodeID = msg.NodeID
	m.stackName = msg.StackName

	m.setRenderItem()

	if m.ready {
		m.List.Viewport.SetContent(m.List.View())
		// Preserve cursor position on refresh, don't call GotoTop
	}
}

func (m *Model) refreshServiceErrorsFromSnapshot() {
	for k := range m.serviceHasError {
		delete(m.serviceHasError, k)
	}
	for k := range m.serviceErrorText {
		delete(m.serviceErrorText, k)
	}

	snap := m.deps.Snapshot.GetSnapshot()
	if snap == nil {
		return
	}

	svcDesired := make(map[string]int)
	svcRunning := make(map[string]int)
	for _, svc := range snap.Services {
		desired := 1
		if svc.Spec.Mode.Replicated != nil && svc.Spec.Mode.Replicated.Replicas != nil {
			desired = int(*svc.Spec.Mode.Replicated.Replicas)
		} else if svc.Spec.Mode.Global != nil {
			desired = len(snap.Nodes)
		}
		svcDesired[svc.ID] = desired
		// Initialize svcRunning with 0 for all services
		svcRunning[svc.ID] = 0
	}

	latestTasks := latestTasksByServiceKey(snap.Tasks)

	for _, t := range latestTasks {
		if t.DesiredState == swarm.TaskStateRunning && t.Status.State == swarm.TaskStateRunning {
			svcRunning[t.ServiceID]++
		}
	}

	for _, t := range latestTasks {
		desired := svcDesired[t.ServiceID]
		running := svcRunning[t.ServiceID]
		if desired == 0 || running >= desired {
			continue
		}
		if t.DesiredState == swarm.TaskStateShutdown && t.Status.State == swarm.TaskStateComplete {
			continue
		}
		hasError := false
		// Only count explicit errors: Status.Err or Failed/Rejected states
		if t.Status.Err != "" {
			hasError = true
		} else if t.Status.State == swarm.TaskStateFailed || t.Status.State == swarm.TaskStateRejected {
			hasError = true
		}
		if !hasError {
			continue
		}
		m.serviceHasError[t.ServiceID] = true
		if m.serviceErrorText[t.ServiceID] == "" {
			m.serviceErrorText[t.ServiceID] = t.Status.Err
		}
	}

	// If service is under-replicated with no explicit error from latest task,
	// check recent tasks for the most recent error
	for serviceID, running := range svcRunning {
		desired := svcDesired[serviceID]
		if desired > 0 && running < desired && m.serviceErrorText[serviceID] == "" {
			// Find most recent task timestamp for this service
			var newestTaskTime time.Time
			for _, t := range snap.Tasks {
				if t.ServiceID != serviceID {
					continue
				}
				at := t.Status.Timestamp
				if at.IsZero() {
					at = t.CreatedAt
				}
				if newestTaskTime.IsZero() || at.After(newestTaskTime) {
					newestTaskTime = at
				}
			}

			// Only check tasks within 5 minutes of the newest task
			cutoff := newestTaskTime.Add(-5 * time.Minute)

			// Find most recent task with an actual error (not just non-running)
			var mostRecentErr string
			var mostRecentErrTime time.Time
			for _, t := range snap.Tasks {
				if t.ServiceID != serviceID {
					continue
				}
				if t.Status.Err == "" {
					continue
				}
				at := t.Status.Timestamp
				if at.IsZero() {
					at = t.CreatedAt
				}
				if at.Before(cutoff) {
					continue
				}
				if mostRecentErr == "" || at.After(mostRecentErrTime) {
					mostRecentErr = t.Status.Err
					mostRecentErrTime = at
				}
			}
			if mostRecentErr != "" {
				m.serviceHasError[serviceID] = true
				m.serviceErrorText[serviceID] = mostRecentErr
			}
		}
	}
}

func latestTasksByServiceKey(tasks []swarm.Task) []swarm.Task {
	latest := make(map[string]swarm.Task)
	latestAt := make(map[string]time.Time)
	for _, t := range tasks {
		key := taskKeyForService(t)
		at := t.Status.Timestamp
		if at.IsZero() {
			at = t.CreatedAt
		}
		if prevAt, ok := latestAt[key]; !ok || at.After(prevAt) {
			latestAt[key] = at
			latest[key] = t
		}
	}

	res := make([]swarm.Task, 0, len(latest))
	for _, t := range latest {
		res = append(res, t)
	}
	return res
}

func taskKeyForService(t swarm.Task) string {
	if t.Slot > 0 {
		return fmt.Sprintf("%s:%d", t.ServiceID, t.Slot)
	}
	return fmt.Sprintf("%s:%s", t.ServiceID, t.NodeID)
}

// computeColWidths centralizes column width calculation so header and rows
// use the exact same sizes. Uses equal division like CONFIG view.
func (m *Model) computeColWidths(width int) []int {
	if width <= 0 {
		width = 80
	}
	cols := 10
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

	// Ensure STATUS and UPDATED columns have at least 10 chars
	minStatus := 10
	minUpdated := 10
	cur := colWidths[3] + colWidths[8]
	if cur < minStatus+minUpdated {
		deficit := minStatus + minUpdated - cur
		for i := 2; i >= 0 && deficit > 0; i-- {
			take := deficit
			if colWidths[i] > take+5 {
				colWidths[i] -= take
				deficit = 0
			} else {
				take = colWidths[i] - 5
				if take > 0 {
					colWidths[i] -= take
					deficit -= take
				}
			}
		}
		if colWidths[3] < minStatus {
			colWidths[3] = minStatus
		}
		if colWidths[8] < minUpdated {
			colWidths[8] = minUpdated
		}
	}

	if colWidths[2] < 1 {
		colWidths[2] = 1
	}
	return colWidths
}

func (m *Model) setRenderItem() {
	// Use shared computation for column widths to keep header and rows in sync
	m.List.RenderItem = func(e docker.ServiceEntry, selected bool, _ int) string {
		colWidths := m.List.ColWidths()
		if len(colWidths) < 10 {
			return e.ServiceName
		}

		// Cache for header alignment
		m.colServiceWidth = colWidths[0]
		m.colStackWidth = colWidths[1]

		// Prepare texts
		replicasText := fmt.Sprintf("%d/%d", e.ReplicasOnNode, e.ReplicasTotal)
		if e.ReplicasTotal == 0 {
			replicasText = "—"
		}

		// For selected row, apply scrolling only to columns that are actually truncated
		var serviceName, stackName, statusText, modeText, imageText, portsText, created, updated, errText string

		if selected {
			// Check each column - only scroll if truncated
			if len(e.ServiceName) > colWidths[0]-1 {
				serviceName = formatErrorWithScroll(e.ServiceName, m.columnScrollOffset, colWidths[0]-1)
			} else {
				serviceName = truncateWithEllipsis(e.ServiceName, colWidths[0]-1)
			}

			if len(e.Image) > colWidths[5] {
				imageText = formatErrorWithScroll(e.Image, m.columnScrollOffset, colWidths[5])
			} else {
				imageText = truncateWithEllipsis(e.Image, colWidths[5])
			}

			if len(e.Ports) > colWidths[6] {
				portsText = formatErrorWithScroll(e.Ports, m.columnScrollOffset, colWidths[6])
			} else {
				portsText = truncateWithEllipsis(e.Ports, colWidths[6])
			}

			errStr := m.serviceErrorText[e.ServiceID]
			if len(errStr) > colWidths[9] {
				errText = formatErrorWithScroll(errStr, m.columnScrollOffset, colWidths[9])
			} else {
				errText = truncateWithEllipsis(errStr, colWidths[9])
			}

			// These columns rarely need scrolling, just truncate
			stackName = truncateWithEllipsis(e.StackName, colWidths[1])
			statusText = truncateWithEllipsis(e.Status, colWidths[3])
			modeText = truncateWithEllipsis(e.Mode, colWidths[4])
			created = truncateWithEllipsis(formatRelativeTime(e.CreatedAt), colWidths[7])
			updated = truncateWithEllipsis(formatRelativeTime(e.UpdatedAt), colWidths[8])
		} else {
			// For non-selected rows, just truncate normally
			serviceName = truncateWithEllipsis(e.ServiceName, colWidths[0]-1)
			stackName = truncateWithEllipsis(e.StackName, colWidths[1])
			statusText = truncateWithEllipsis(e.Status, colWidths[3])
			modeText = truncateWithEllipsis(e.Mode, colWidths[4])
			imageText = truncateWithEllipsis(e.Image, colWidths[5])
			portsText = truncateWithEllipsis(e.Ports, colWidths[6])
			created = truncateWithEllipsis(formatRelativeTime(e.CreatedAt), colWidths[7])
			updated = truncateWithEllipsis(formatRelativeTime(e.UpdatedAt), colWidths[8])
			errText = truncateWithEllipsis(m.serviceErrorText[e.ServiceID], colWidths[9])
		}

		itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))

		// Build format string with all columns at once (like CONFIG view)
		formatStr := fmt.Sprintf(" %%-%ds%%-%ds%%-%ds%%-%ds%%-%ds%%-%ds%%-%ds%%-%ds%%-%ds%%-%ds",
			colWidths[0]-1, colWidths[1], colWidths[2], colWidths[3], colWidths[4],
			colWidths[5], colWidths[6], colWidths[7], colWidths[8], colWidths[9])

		var lineStr string
		if selected && m.selectedTaskIndex == -1 {
			// Only highlight service row if no task is selected
			selBg := lipgloss.Color("25") // Lighter blue
			selBase := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(selBg).Bold(true)
			lineStr = selBase.Render(fmt.Sprintf(formatStr,
				serviceName, stackName, replicasText, statusText, modeText, imageText, portsText, created, updated, errText))

			// Ensure highlight background fills the full viewport width
			if m.List.Viewport.Width > 0 {
				w := lipgloss.Width(lineStr)
				if m.List.Viewport.Width > w {
					pad := m.List.Viewport.Width - w
					lineStr += selBase.Render(strings.Repeat(" ", pad))
				}
			}
		} else if m.serviceHasError[e.ServiceID] {
			// Color non-selected error rows red
			errStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
			lineStr = errStyle.Render(fmt.Sprintf(formatStr,
				serviceName, stackName, replicasText, statusText, modeText, imageText, portsText, created, updated, errText))
		} else {
			// Normal rendering
			lineStr = itemStyle.Render(fmt.Sprintf(formatStr,
				serviceName, stackName, replicasText, statusText, modeText, imageText, portsText, created, updated, errText))
		}

		// Check if service is expanded and add task rows
		if m.expandedServices[e.ServiceID] {
			tasks := m.serviceTasks[e.ServiceID]
			if len(tasks) > 0 {
				// Add task header (include ERROR column)
				taskHeaderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
				// Align header columns with task row formatting
				taskHeader := fmt.Sprintf("   %-22s  %-12s  %-13s  %-40s  %s",
					"NAME", "NODE", "DESIRED STATE", "CURRENT STATE", "ERROR")
				lineStr += "\n" + taskHeaderStyle.Render(taskHeader)

				// Add each task as a row
				for taskIdx, task := range tasks {
					taskName := truncateWithEllipsis(task.Name, 22)
					taskNode := truncateWithEllipsis(task.NodeName, 12)
					taskDesired := truncateWithEllipsis(task.DesiredState, 13)
					taskCurrent := truncateWithEllipsis(task.CurrentState, 40)
					taskErr := truncateWithEllipsis(task.Error, 30)

					// Check if this task is selected
					taskSelected := selected && m.selectedTaskIndex == taskIdx
					var taskLine string
					if taskSelected {
						// Lighter highlight for task rows
						taskSelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24")).Bold(true)
						taskLine = taskSelStyle.Render(fmt.Sprintf("   %-22s  %-12s  %-13s  %-40s  %s",
							taskName, taskNode, taskDesired, taskCurrent, taskErr))
						// Pad task highlight to full width
						if m.List.Viewport.Width > 0 {
							tw := lipgloss.Width(taskLine)
							if m.List.Viewport.Width > tw {
								pad := m.List.Viewport.Width - tw
								taskLine += taskSelStyle.Render(strings.Repeat(" ", pad))
							}
						}
					} else {
						taskStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
						taskLine = taskStyle.Render(fmt.Sprintf("   %-22s  %-12s  %-13s  %-40s  %s",
							taskName, taskNode, taskDesired, taskCurrent, taskErr))
					}
					lineStr += "\n" + taskLine
				}
			} else {
				// Show "no tasks" message
				noTasksStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
				lineStr += "\n" + noTasksStyle.Render("   (no tasks)")
			}
		}

		return lineStr
	}
}

// formatRelativeTime formats a time as a relative duration (e.g., "2h ago", "3d ago")
func formatRelativeTime(t time.Time) string {
	if t.IsZero() {
		return "-"
	}

	d := time.Since(t)
	if d < time.Minute {
		return "just now"
	} else if d < time.Hour {
		mins := int(d.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		return fmt.Sprintf("%dh ago", hours)
	} else if d < 7*24*time.Hour {
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	} else if d < 30*24*time.Hour {
		weeks := int(d.Hours() / 24 / 7)
		return fmt.Sprintf("%dw ago", weeks)
	} else if d < 365*24*time.Hour {
		months := int(d.Hours() / 24 / 30)
		return fmt.Sprintf("%dmo ago", months)
	} else {
		years := int(d.Hours() / 24 / 365)
		return fmt.Sprintf("%dy ago", years)
	}
}

// GetServicesHelpContent returns categorized help for the services view
func GetServicesHelpContent() []helpview.HelpCategory {
	return []helpview.HelpCategory{
		{
			Title: "General",
			Items: []helpview.HelpItem{
				{Keys: "<i>", Description: "Inspect service"},
				{Keys: "<p>", Description: "Show/hide tasks"},
				{Keys: "<l>", Description: "View logs"},
				{Keys: "<s>", Description: "Scale service"},
				{Keys: "<r>", Description: "Restart service"},
				{Keys: "<ctrl+r>", Description: "Rollback service"},
				{Keys: "<ctrl+d>", Description: "Remove service"},
				{Keys: "<x>", Description: view.BEHelpDesc("shell", "Open shell into service container")},
				{Keys: "<w>", Description: view.BEHelpDesc("port-forward", "Forward service ports to localhost")},
				{Keys: "</>", Description: "Filter"},
			},
		},
		{
			Title: "View",
			Items: []helpview.HelpItem{
				{Keys: "<shift+n>", Description: "Order by Name"},
				{Keys: "<shift+s>", Description: "Order by Status"},
				{Keys: "<shift+i>", Description: "Order by Image"},
				{Keys: "<shift+p>", Description: "Order by Ports"},
				{Keys: "<shift+c>", Description: "Order by Created"},
				{Keys: "<shift+u>", Description: "Order by Updated"},
				{Keys: "<shift+r>", Description: "Order by Error"},
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

// applySorting applies the current sort configuration to the filtered list
func (m *Model) applySorting() {
	if len(m.List.Filtered) == 0 {
		return
	}

	// Remember cursor position
	cursorService := ""
	if m.List.Cursor < len(m.List.Filtered) {
		cursorService = m.List.Filtered[m.List.Cursor].ServiceName
	}

	// Sort the filtered list
	switch m.sortField {
	case SortByName:
		sort.Slice(m.List.Filtered, func(i, j int) bool {
			if m.sortAscending {
				return m.List.Filtered[i].ServiceName < m.List.Filtered[j].ServiceName
			}
			return m.List.Filtered[i].ServiceName > m.List.Filtered[j].ServiceName
		})
	case SortByStatus:
		sort.Slice(m.List.Filtered, func(i, j int) bool {
			if m.sortAscending {
				return m.List.Filtered[i].Status < m.List.Filtered[j].Status
			}
			return m.List.Filtered[i].Status > m.List.Filtered[j].Status
		})
	case SortByImage:
		sort.Slice(m.List.Filtered, func(i, j int) bool {
			if m.sortAscending {
				return m.List.Filtered[i].Image < m.List.Filtered[j].Image
			}
			return m.List.Filtered[i].Image > m.List.Filtered[j].Image
		})
	case SortByPorts:
		sort.Slice(m.List.Filtered, func(i, j int) bool {
			if m.sortAscending {
				return m.List.Filtered[i].Ports < m.List.Filtered[j].Ports
			}
			return m.List.Filtered[i].Ports > m.List.Filtered[j].Ports
		})
	case SortByCreated:
		sort.Slice(m.List.Filtered, func(i, j int) bool {
			if m.sortAscending {
				return m.List.Filtered[i].CreatedAt.Before(m.List.Filtered[j].CreatedAt)
			}
			return m.List.Filtered[i].CreatedAt.After(m.List.Filtered[j].CreatedAt)
		})
	case SortByUpdated:
		sort.Slice(m.List.Filtered, func(i, j int) bool {
			if m.sortAscending {
				return m.List.Filtered[i].UpdatedAt.Before(m.List.Filtered[j].UpdatedAt)
			}
			return m.List.Filtered[i].UpdatedAt.After(m.List.Filtered[j].UpdatedAt)
		})
	case SortByError:
		sort.Slice(m.List.Filtered, func(i, j int) bool {
			iHas := m.serviceHasError[m.List.Filtered[i].ServiceID]
			jHas := m.serviceHasError[m.List.Filtered[j].ServiceID]
			if iHas == jHas {
				iText := m.serviceErrorText[m.List.Filtered[i].ServiceID]
				jText := m.serviceErrorText[m.List.Filtered[j].ServiceID]
				if iText == jText {
					if m.sortAscending {
						return m.List.Filtered[i].ServiceName < m.List.Filtered[j].ServiceName
					}
					return m.List.Filtered[i].ServiceName > m.List.Filtered[j].ServiceName
				}
				if m.sortAscending {
					return iText < jText
				}
				return iText > jText
			}
			if m.sortAscending {
				return iHas && !jHas
			}
			return !iHas && jHas
		})
	}

	// Restore cursor position to previously selected item
	if cursorService != "" {
		for i, s := range m.List.Filtered {
			if s.ServiceName == cursorService {
				m.List.Cursor = i
				return
			}
		}
	}

	// If previous item not found, reset cursor to 0
	m.List.Cursor = 0
	m.List.Viewport.GotoTop()
}
