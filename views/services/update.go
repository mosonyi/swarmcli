package servicesview

import (
	"context"
	"fmt"
	"strings"
	"swarmcli/core/primitives/hash"
	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/confirmdialog"
	inspectview "swarmcli/views/inspect"
	logsview "swarmcli/views/logs"
	"swarmcli/views/scaledialog"
	"swarmcli/views/view"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {

	case Msg:
		l().Infof("ServicesView: Received Msg with %d entries", len(msg.Entries))

		// If we're viewing a specific stack or node and all services are gone, go back to stacks view
		// This handles cases where:
		// - All services in a stack are deleted (stack no longer exists)
		// - All services on a node are removed
		// The stacks view will automatically refresh since we already called RefreshSnapshot
		// Use Replace=true to clear navigation history so ESC doesn't come back here
		if len(msg.Entries) == 0 && (msg.FilterType == StackFilter || msg.FilterType == NodeFilter) {
			l().Info("ServicesView: No services remaining in filtered view, navigating back to stacks")
			m.Visible = false
			return func() tea.Msg {
				return view.NavigateToMsg{ViewName: "stacks", Payload: nil, Replace: true}
			}
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
		// Continue polling
		return tickCmd()

	case TickMsg:
		l().Infof("ServicesView: Received TickMsg, visible=%v", m.Visible)
		// Check for changes (this will return either a Msg or the next TickMsg)
		if m.Visible {
			return CheckServicesCmd(m.lastSnapshot, m.filterType, m.nodeID, m.stackName)
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
			return func() tea.Msg {
				if err := docker.ScaleService(entry.ServiceID, msg.Replicas); err != nil {
					l().Errorf("Failed to scale service %s: %v", entry.ServiceName, err)
					return ScaleErrorMsg{
						ServiceName: entry.ServiceName,
						Error:       err,
					}
				}
				l().Infof("Successfully scaled service %s to %d replicas", entry.ServiceName, msg.Replicas)
				// Force immediate snapshot refresh
				if _, err := docker.RefreshSnapshot(); err != nil {
					l().Warnf("Failed to refresh snapshot: %v", err)
				}
				return refreshServicesCmd(m.nodeID, m.stackName, m.filterType)()
			}
		}
		return nil

	case confirmdialog.ResultMsg:
		m.confirmDialog.Visible = false

		if msg.Confirmed && m.List.Cursor < len(m.List.Filtered) {
			entry := m.List.Filtered[m.List.Cursor]

			switch m.pendingAction {
			case "remove":
				l().Debugln("Starting remove for", entry.ServiceName)
				return func() tea.Msg {
					l().Infof("Executing remove for service: %s", entry.ServiceName)
					if err := docker.RemoveService(entry.ServiceName); err != nil {
						l().Errorf("Failed to remove service %s: %v", entry.ServiceName, err)
						return RemoveErrorMsg{
							ServiceName: entry.ServiceName,
							Error:       err,
						}
					}
					l().Infof("Successfully removed service: %s", entry.ServiceName)
					// Force immediate snapshot refresh
					if _, err := docker.RefreshSnapshot(); err != nil {
						l().Warnf("Failed to refresh snapshot: %v", err)
					}
					return refreshServicesCmd(m.nodeID, m.stackName, m.filterType)()
				}
			case "rollback":
				l().Debugln("Starting rollback for", entry.ServiceName)
				return func() tea.Msg {
					l().Infof("Executing rollback for service: %s", entry.ServiceName)
					if err := docker.RollbackService(entry.ServiceName); err != nil {
						l().Errorf("Failed to rollback service %s: %v", entry.ServiceName, err)
						return RollbackErrorMsg{
							ServiceName: entry.ServiceName,
							Error:       err,
						}
					}
					l().Infof("Successfully rolled back service: %s", entry.ServiceName)
					// Force immediate snapshot refresh
					if _, err := docker.RefreshSnapshot(); err != nil {
						l().Warnf("Failed to refresh snapshot: %v", err)
					}
					return refreshServicesCmd(m.nodeID, m.stackName, m.filterType)()
				}
			default:
				// Default to restart
				l().Debugln("Starting restart for", entry.ServiceName)
				return func() tea.Msg {
					l().Infof("Executing restart for service: %s", entry.ServiceName)
					if err := docker.RestartService(entry.ServiceName); err != nil {
						l().Errorf("Failed to restart service %s: %v", entry.ServiceName, err)
						return RestartErrorMsg{
							ServiceName: entry.ServiceName,
							Error:       err,
						}
					}
					l().Infof("Successfully restarted service: %s", entry.ServiceName)
					// Force immediate snapshot refresh
					if _, err := docker.RefreshSnapshot(); err != nil {
						l().Warnf("Failed to refresh snapshot: %v", err)
					}
					return refreshServicesCmd(m.nodeID, m.stackName, m.filterType)()
				}
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
					return func() tea.Msg {
						tasks, err := docker.GetTasksForService(entry.ServiceID)
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
				return func() tea.Msg {
					content, err := docker.Inspect(context.Background(), docker.InspectService, entry.ServiceID)
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
		case "q":
			m.Visible = false
			// Go back to stacks view
			return func() tea.Msg { return view.NavigateToMsg{ViewName: "stacks", Payload: nil} }

		case "esc":
			// ESC should also go back to stacks view
			m.Visible = false
			return func() tea.Msg { return view.NavigateToMsg{ViewName: "stacks", Payload: nil} }
		}

		m.List.Viewport.SetContent(m.List.View())
		return nil
	}

	var cmd tea.Cmd
	m.List.Viewport, cmd = m.List.Viewport.Update(msg)
	return cmd
}

func (m *Model) SetContent(msg Msg) {
	l().Infof("ServicesView.SetContent: Updating display with %d services", len(msg.Entries))

	m.title = msg.Title

	m.List.Items = msg.Entries
	m.List.ApplyFilter()

	m.filterType = msg.FilterType
	m.nodeID = msg.NodeID
	m.stackName = msg.StackName

	m.setRenderItem()

	if m.ready {
		m.List.Viewport.SetContent(m.List.View())
		// Preserve cursor position on refresh, don't call GotoTop
	}
}

func (m *Model) setRenderItem() {
	// We'll compute column widths based on the longest service name then
	// allocate the remaining space to the other columns.
	m.List.RenderItem = func(e docker.ServiceEntry, selected bool, _ int) string {
		width := m.List.Viewport.Width
		if width <= 0 {
			width = 80
		}

		cols := 9
		sepLen := 2
		sepTotal := sepLen * (cols - 1)
		// Effective width available for columns (excluding separators)
		effWidth := width - sepTotal
		if effWidth < cols { // ensure sensible minimum
			effWidth = width
		}
		colWidths := make([]int, cols)

		// Headers and sensible minimums
		headers := []string{" SERVICE", "STACK", "REPLICAS", "STATUS", "MODE", "IMAGE", "PORTS", "CREATED", "UPDATED"}
		minCols := make([]int, cols)
		for i := 0; i < cols; i++ {
			hw := lipgloss.Width(headers[i])
			floor := 6
			switch i {
			case 0: // SERVICE
				floor = 10
			case 1: // STACK
				floor = 10
			case 2: // REPLICAS
				floor = 8
			case 3: // STATUS
				floor = 8
			case 4: // MODE
				floor = 10
			case 5: // IMAGE
				floor = 15
			case 6: // PORTS
				floor = 8
			case 7, 8: // CREATED, UPDATED
				floor = 8
			}
			if hw > floor {
				minCols[i] = hw
			} else {
				minCols[i] = floor
			}
		}

		// Find longest service name visually among items (include header)
		maxSvc := lipgloss.Width(headers[0])
		for _, it := range m.List.Items {
			if s, ok := any(it).(docker.ServiceEntry); ok {
				if w := lipgloss.Width(s.ServiceName); w > maxSvc {
					maxSvc = w
				}
			}
		}
		desiredSvc := maxSvc + 1 // reserve 1 for leading space

		// Assign desired service width if possible, otherwise use proportional
		// partitioning but ensure minimums.
		nonServiceMinSum := 0
		for i := 1; i < cols; i++ {
			nonServiceMinSum += minCols[i]
		}

		if desiredSvc+nonServiceMinSum <= effWidth {
			colWidths[0] = desiredSvc
			// Start with minimums for remaining columns
			for i := 1; i < cols; i++ {
				colWidths[i] = minCols[i]
			}
			// Distribute leftover space equally among columns 1..5 (within effWidth)
			sum := 0
			for _, v := range colWidths {
				sum += v
			}
			leftover := effWidth - sum
			if leftover > 0 {
				per := leftover / (cols - 1)
				rem := leftover % (cols - 1)
				for i := 1; i < cols; i++ {
					add := per
					if rem > 0 {
						add++
						rem--
					}
					colWidths[i] += add
				}
			}
		} else {
			// Proportional partition but respect minimums
			// Partition across the effective width
			base := effWidth / cols
			for i := 0; i < cols; i++ {
				colWidths[i] = base
			}
			for i := 0; i < cols; i++ {
				if colWidths[i] < minCols[i] {
					colWidths[i] = minCols[i]
				}
			}
			// Adjust last column to ensure total equals effWidth
			sum := 0
			for _, v := range colWidths {
				sum += v
			}
			if sum != effWidth {
				colWidths[cols-1] += effWidth - sum
				if colWidths[cols-1] < 1 {
					colWidths[cols-1] = 1
				}
			}
		}

		// Cache for header alignment
		m.colServiceWidth = colWidths[0]
		m.colStackWidth = colWidths[1]

		// Prepare texts
		replicasText := fmt.Sprintf("%d/%d", e.ReplicasOnNode, e.ReplicasTotal)
		if e.ReplicasTotal == 0 {
			replicasText = "—"
		}

		// Truncate columns except the first (we try not to shorten first column
		// but if it still doesn't fit due to small viewport, fall back to
		// truncation there too).
		lastIdx := len(colWidths) - 1
		svcTruncWidth := colWidths[0]
		// If we reserved a leading space in formatting, reduce by 1
		if svcTruncWidth > 0 {
			svcTruncWidth = svcTruncWidth - 1
		}
		serviceName := truncateWithEllipsis(e.ServiceName, svcTruncWidth)
		stackName := truncateWithEllipsis(e.StackName, colWidths[1]-1)
		statusText := truncateWithEllipsis(e.Status, colWidths[3]-1)
		modeText := truncateWithEllipsis(e.Mode, colWidths[4]-1)
		imageText := truncateWithEllipsis(e.Image, colWidths[5]-1)
		portsText := truncateWithEllipsis(e.Ports, colWidths[6]-1)
		created := truncateWithEllipsis(formatRelativeTime(e.CreatedAt), colWidths[7]-1)
		updated := truncateWithEllipsis(formatRelativeTime(e.UpdatedAt), colWidths[lastIdx])

		itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
		col0 := itemStyle.Render(fmt.Sprintf(" %-*s", colWidths[0]-1, serviceName))
		col1 := itemStyle.Render(fmt.Sprintf("%-*s", colWidths[1]-1, stackName))

		var replicasColor lipgloss.Color
		switch {
		case e.ReplicasTotal == 0:
			replicasColor = lipgloss.Color("8")
		case e.ReplicasOnNode == 0:
			replicasColor = lipgloss.Color("9")
		case e.ReplicasOnNode < e.ReplicasTotal:
			replicasColor = lipgloss.Color("11")
		default:
			replicasColor = lipgloss.Color("10")
		}
		replicasStyle := lipgloss.NewStyle().Foreground(replicasColor)
		col2 := replicasStyle.Render(fmt.Sprintf("%-*s", colWidths[2]-1, replicasText))

		statusColor := getStatusColor(e.Status)
		statusStyle := lipgloss.NewStyle().Foreground(statusColor)
		col3 := statusStyle.Render(fmt.Sprintf("%-*s", colWidths[3]-1, statusText))

		col4 := itemStyle.Render(fmt.Sprintf("%-*s", colWidths[4]-1, modeText))
		col5 := itemStyle.Render(fmt.Sprintf("%-*s", colWidths[5]-1, imageText))
		col6 := itemStyle.Render(fmt.Sprintf("%-*s", colWidths[6]-1, portsText))
		col7 := itemStyle.Render(fmt.Sprintf("%-*s", colWidths[7]-1, created))
		col8 := itemStyle.Render(fmt.Sprintf("%-*s", colWidths[8], updated))

		// Join with two-space separators for readability
		sep := strings.Repeat(" ", sepLen)
		line := col0 + sep + col1 + sep + col2 + sep + col3 + sep + col4 + sep + col5 + sep + col6 + sep + col7 + sep + col8

		if selected && m.selectedTaskIndex == -1 {
			// Only highlight service row if no task is selected
			selBg := lipgloss.Color("25") // Lighter blue
			selBase := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(selBg).Bold(true)
			selRep := lipgloss.NewStyle().Foreground(replicasColor).Background(selBg).Bold(true)
			selStatus := lipgloss.NewStyle().Foreground(statusColor).Background(selBg).Bold(true)

			// Render each styled column including the separator so the
			// highlight background is continuous across the whole line.
			sepStr := strings.Repeat(" ", sepLen)
			col0 = selBase.Render(fmt.Sprintf(" %-*s", colWidths[0]-1, serviceName) + sepStr)
			col1 = selBase.Render(fmt.Sprintf("%-*s", colWidths[1]-1, stackName) + sepStr)
			col2 = selRep.Render(fmt.Sprintf("%-*s", colWidths[2]-1, replicasText) + sepStr)
			col3 = selStatus.Render(fmt.Sprintf("%-*s", colWidths[3]-1, statusText) + sepStr)
			col4 = selBase.Render(fmt.Sprintf("%-*s", colWidths[4]-1, modeText) + sepStr)
			col5 = selBase.Render(fmt.Sprintf("%-*s", colWidths[5]-1, imageText) + sepStr)
			col6 = selBase.Render(fmt.Sprintf("%-*s", colWidths[6]-1, portsText) + sepStr)
			col7 = selBase.Render(fmt.Sprintf("%-*s", colWidths[7]-1, created) + sepStr)
			col8 = selBase.Render(fmt.Sprintf("%-*s", colWidths[8], updated))
			line = col0 + col1 + col2 + col3 + col4 + col5 + col6 + col7 + col8
		}

		// Check if service is expanded and add task rows
		if m.expandedServices[e.ServiceID] {
			tasks := m.serviceTasks[e.ServiceID]
			if len(tasks) > 0 {
				// Add task header
				taskHeaderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
				taskHeader := taskHeaderStyle.Render("   NAME                    NODE          DESIRED STATE  CURRENT STATE")
				line += "\n" + taskHeader

				// Add each task as a row
				for taskIdx, task := range tasks {
					taskName := truncateWithEllipsis(task.Name, 22)
					taskNode := truncateWithEllipsis(task.NodeName, 12)
					taskDesired := truncateWithEllipsis(task.DesiredState, 13)
					taskCurrent := truncateWithEllipsis(task.CurrentState, 50)

					// Check if this task is selected
					taskSelected := selected && m.selectedTaskIndex == taskIdx
					var taskLine string
					if taskSelected {
						// Lighter highlight for task rows
						taskSelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24")).Bold(true)
						taskLine = taskSelStyle.Render(fmt.Sprintf("   %-22s  %-12s  %-13s  %s",
							taskName, taskNode, taskDesired, taskCurrent))
					} else {
						taskStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
						taskLine = taskStyle.Render(fmt.Sprintf("   %-22s  %-12s  %-13s  %s",
							taskName, taskNode, taskDesired, taskCurrent))
					}
					line += "\n" + taskLine
				}
			} else {
				// Show "no tasks" message
				noTasksStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
				line += "\n" + noTasksStyle.Render("   (no tasks)")
			}
		}

		return line
	}
}

func truncateWithEllipsis(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	// A maxWidth of 1 should render as the ellipsis
	if maxWidth <= 1 {
		return "…"
	}

	// If the string already fits in the available visual width, return it.
	if lipgloss.Width(s) <= maxWidth {
		return s
	}

	// Reserve width for the ellipsis
	ell := "…"
	ellW := lipgloss.Width(ell)

	// Build up runes until adding the next would exceed maxWidth-ellW
	var outRunes []rune
	cur := ""
	for _, r := range s {
		cur += string(r)
		if lipgloss.Width(cur)+ellW > maxWidth {
			break
		}
		outRunes = append(outRunes, r)
	}
	if len(outRunes) == 0 {
		return ell
	}
	return string(outRunes) + ell
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

// getStatusColor returns the appropriate color for a service status
func getStatusColor(status string) lipgloss.Color {
	switch status {
	case "updating", "rolling back":
		return lipgloss.Color("11") // yellow
	case "updated", "active":
		return lipgloss.Color("10") // green
	case "paused", "rollback paused":
		return lipgloss.Color("8") // gray
	case "rolled back":
		return lipgloss.Color("9") // red
	default:
		return lipgloss.Color("15") // white
	}
}
