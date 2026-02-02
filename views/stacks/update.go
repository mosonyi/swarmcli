// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package stacksview

import (
	"fmt"
	"runtime/debug"
	"sort"
	"strings"
	"swarmcli/core/primitives/hash"
	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/confirmdialog"
	helpview "swarmcli/views/help"
	servicesview "swarmcli/views/services"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Update handles all messages for the stacks view.
func (m *Model) Update(msg tea.Msg) tea.Cmd {
	defer func() {
		if r := recover(); r != nil {
			l().Errorf("panic in Stacks.Update: %v", r)
			l().Errorf("%s", debug.Stack())
		}
	}()
	switch msg := msg.(type) {

	case Msg:
		l().Infof("[update]: Received Msg with %d entries", len(msg.Stacks))
		// Update the hash with new data
		var err error
		m.lastSnapshot, err = hash.Compute(msg.Stacks)
		if err != nil {
			l().Errorf("[update] Error computing hash: %v", err)
			return nil
		}
		m.nodeID = msg.NodeID
		m.setStacks(msg.Stacks)
		m.Visible = true

		// Start background fetches for stacks to surface errors without pressing 'p'.
		// Create commands that will load tasks for each stack asynchronously.
		var cmds []tea.Cmd
		// Always keep the tick running
		cmds = append(cmds, tickCmd())
		for _, s := range msg.Stacks {
			stackName := s.Name
			// If we already have tasks cached, skip
			if _, ok := m.stackTasks[stackName]; ok {
				continue
			}
			// Launch async fetch for this stack
			cmds = append(cmds, func(name string) tea.Cmd {
				return func() tea.Msg {
					tasks, err := docker.GetTasksForStack(name)
					return StackTasksLoadedMsg{StackName: name, Tasks: tasks, Error: err}
				}
			}(stackName))
		}
		return tea.Batch(cmds...)

	case TickMsg:
		l().Infof("StacksView: Received TickMsg, visible=%v", m.Visible)
		// Check for changes (this will return either a Msg or the next TickMsg)
		if m.Visible {
			return CheckStacksCmd(m.lastSnapshot, m.nodeID)
		}
		// Continue polling even if not visible
		return tickCmd()

	case RefreshErrorMsg:
		m.Visible = true
		m.List.Viewport.SetContent(fmt.Sprintf("Error refreshing stacks: %v", msg.Err))
		return nil

	case StackTasksLoadedMsg:
		// Store loaded tasks for the stack and re-render
		if msg.Error != nil {
			l().Errorf("Failed to fetch tasks for stack %s: %v", msg.StackName, msg.Error)
			m.stackTasks[msg.StackName] = []docker.TaskEntry{}
		} else {
			m.stackTasks[msg.StackName] = msg.Tasks
		}
		// If the stack is expanded and currently selected, default into first task
		if m.expandedStacks[msg.StackName] {
			// If the currently focused item is this stack and we have tasks, select first
			if m.List.Cursor < len(m.List.Filtered) && m.List.Filtered[m.List.Cursor].Name == msg.StackName {
				if len(m.stackTasks[msg.StackName]) > 0 {
					m.selectedTaskIndex = 0
				}
			}
		}
		m.setRenderItem()
		return nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.List.Viewport.Width = msg.Width
		m.List.Viewport.Height = msg.Height
		m.ready = true

		// On first resize (initialization), always reset YOffset to 0
		// This fixes the issue where the view is created with small dimensions,
		// then resized, causing YOffset to be incorrectly set
		if m.firstResize {
			m.List.Viewport.YOffset = 0
			m.firstResize = false
			l().Info("First WindowSizeMsg: resetting YOffset to 0")
		} else if m.List.Cursor == 0 {
			// On subsequent resizes, only reset YOffset if cursor is at top
			m.List.Viewport.YOffset = 0
		}
		return nil

	case confirmdialog.ResultMsg:
		m.confirmDialog.Visible = false

		if msg.Confirmed && m.List.Cursor < len(m.List.Filtered) {
			selected := m.List.Filtered[m.List.Cursor]

			if m.pendingAction == "remove" {
				l().Debugln("Starting remove for stack", selected.Name)
				return func() tea.Msg {
					l().Infof("Executing remove for stack: %s", selected.Name)
					if err := docker.RemoveStack(selected.Name); err != nil {
						l().Errorf("Failed to remove stack %s: %v", selected.Name, err)
						return RemoveErrorMsg{
							StackName: selected.Name,
							Error:     err,
						}
					}
					l().Infof("Successfully removed stack: %s", selected.Name)
					// Force immediate snapshot refresh
					if _, err := docker.RefreshSnapshot(); err != nil {
						l().Warnf("Failed to refresh snapshot: %v", err)
					}
					return CheckStacksCmd(m.lastSnapshot, m.nodeID)()
				}
			}
		}
		m.pendingAction = ""
		return nil

	case RemoveErrorMsg:
		m.confirmDialog.Visible = true
		m.confirmDialog.ErrorMode = true
		m.confirmDialog.Message = fmt.Sprintf("Failed to remove stack %q:\n%v", msg.StackName, msg.Error)
		return nil

	case tea.KeyMsg:
		// If confirm dialog is visible, let it handle the key
		if m.confirmDialog.Visible {
			return m.confirmDialog.Update(msg)
		}

		// --- if in search mode, handle all keys via FilterableList ---
		if m.List.Mode == filterlist.ModeSearching {
			m.List.HandleKey(msg)
			return nil
		}

		// --- normal mode ---
		// If ESC is pressed and there's an active filter, clear it instead of quitting
		if msg.Type == tea.KeyEsc && m.List.Query != "" {
			m.List.Query = ""
			m.List.Mode = filterlist.ModeNormal
			m.List.ApplyFilter()
			m.List.Cursor = 0
			m.List.Viewport.GotoTop()
			return nil
		}

		// Note: task navigation is handled below to allow entering/exiting
		// the inline task list with up/down keys. We call HandleKey later
		// after giving task-navigation a chance to intercept the key.

		// Show help screen
		if msg.String() == "?" {
			return func() tea.Msg {
				return view.NavigateToMsg{
					ViewName: view.NameHelp,
					Payload:  GetStacksHelpContent(),
				}
			}
		}

		// Enter triggers navigation to services
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

		// 'p' shows tasks for selected stack
		if msg.String() == "p" {
			if m.List.Cursor < len(m.List.Filtered) {
				selected := m.List.Filtered[m.List.Cursor]
				// Toggle expanded state for this stack to show inline tasks
				m.expandedStacks[selected.Name] = !m.expandedStacks[selected.Name]
				if m.expandedStacks[selected.Name] {
					// Load tasks for this stack asynchronously
					stackName := selected.Name
					return func() tea.Msg {
						tasks, err := docker.GetTasksForStack(stackName)
						return StackTasksLoadedMsg{StackName: stackName, Tasks: tasks, Error: err}
					}
				} else {
					// collapsing: keep cached tasks (don't delete) to preserve error state, just reset selection
					m.selectedTaskIndex = -1
					m.setRenderItem()
				}
			}
		}

		// 'ctrl+d' removes selected stack
		if msg.String() == "ctrl+d" {
			if m.List.Cursor < len(m.List.Filtered) {
				selected := m.List.Filtered[m.List.Cursor]
				m.pendingAction = "remove"
				m.confirmDialog.Visible = true
				m.confirmDialog.ErrorMode = false
				m.confirmDialog.Message = fmt.Sprintf("Remove stack %q?\n\nThis will remove all services in the stack.\nThis action cannot be undone!", selected.Name)
			}
		}

		// Handle task navigation for expanded stacks (up/down into tasks)
		if m.List.Cursor < len(m.List.Filtered) {
			entry := m.List.Filtered[m.List.Cursor]
			if m.expandedStacks[entry.Name] {
				// Distinguish between "tasks not yet loaded" and "loaded but empty".
				tasks, loaded := m.stackTasks[entry.Name]
				switch msg.String() {
				case "down":
					// If tasks not yet loaded, enter task-selection and wait for load
					if !loaded {
						m.selectedTaskIndex = 0
						m.setRenderItem()
						return nil
					}
					if m.selectedTaskIndex < len(tasks)-1 {
						m.selectedTaskIndex++
						m.setRenderItem()
						return nil
					} else if m.selectedTaskIndex == len(tasks)-1 {
						// At last task, move to next stack
						m.selectedTaskIndex = -1
						m.List.HandleKey(msg)
						return nil
					}
				case "up":
					if !loaded {
						// nothing loaded yet, just keep focus on stack row
						m.selectedTaskIndex = -1
						m.setRenderItem()
						return nil
					}
					if m.selectedTaskIndex > 0 {
						m.selectedTaskIndex--
						m.setRenderItem()
						return nil
					} else if m.selectedTaskIndex == 0 {
						// Move back to stack row
						m.selectedTaskIndex = -1
						m.setRenderItem()
						return nil
					}
				}
			} else if msg.String() == "up" && m.selectedTaskIndex == -1 && m.List.Cursor > 0 {
				prevEntry := m.List.Filtered[m.List.Cursor-1]
				if m.expandedStacks[prevEntry.Name] {
					prevTasks := m.stackTasks[prevEntry.Name]
					if len(prevTasks) > 0 {
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

		// Reset task selection and error scroll when moving to different stack
		if oldCursor != m.List.Cursor {
			m.selectedTaskIndex = -1
			m.errorScrollOffset = 0
		}

		// Horizontal scrolling for error messages (like services view)
		if msg.String() == "left" {
			if m.errorScrollOffset > 0 {
				m.errorScrollOffset -= 5
				if m.errorScrollOffset < 0 {
					m.errorScrollOffset = 0
				}
			}
			return nil
		}
		if msg.String() == "right" {
			// Only scroll if current stack has an error
			if m.List.Cursor < len(m.List.Filtered) {
				entry := m.List.Filtered[m.List.Cursor]
				if tasks, ok := m.stackTasks[entry.Name]; ok {
					for _, t := range tasks {
						if strings.ToLower(t.DesiredState) == "running" && t.Error != "" {
							m.errorScrollOffset += 5
							break
						}
					}
				}
			}
			return nil
		}

		// Sort by Stack name (Shift+S)
		if msg.String() == "S" {
			m.userSetSort = true
			if m.sortField == SortByName {
				// Toggle ascending/descending
				m.sortAscending = !m.sortAscending
			} else {
				// Switch to this field and reset to ascending
				m.sortField = SortByName
				m.sortAscending = true
			}
			m.applySorting()
			return nil
		}

		// Sort by Error (Shift+R)
		if msg.String() == "R" {
			m.userSetSort = true
			if m.sortField == SortByError {
				m.sortAscending = !m.sortAscending
			} else {
				m.sortField = SortByError
				m.sortAscending = true
			}
			m.applySorting()
			return nil
		}

		// Sort by Services (Shift+E)
		if msg.String() == "E" {
			m.userSetSort = true
			if m.sortField == SortByServices {
				// Toggle ascending/descending
				m.sortAscending = !m.sortAscending
			} else {
				// Switch to this field and reset to ascending
				m.sortField = SortByServices
				m.sortAscending = true
			}
			m.applySorting()
			return nil
		}

		// Sort by Tasks (Shift+T)
		if msg.String() == "T" {
			m.userSetSort = true
			if m.sortField == SortByTasks {
				// Toggle ascending/descending
				m.sortAscending = !m.sortAscending
			} else {
				// Switch to this field and reset to ascending
				m.sortField = SortByTasks
				m.sortAscending = true
			}
			m.applySorting()
			return nil
		}

		return nil
	}

	var cmd tea.Cmd
	m.List.Viewport, cmd = m.List.Viewport.Update(msg)
	return cmd
}

func (m *Model) setStacks(stacks []docker.StackEntry) {
	l().Infof("StacksView.setStacks: Updating display with %d stacks", len(stacks))

	// Preserve filter query and cursor position
	oldQuery := m.List.Query
	oldMode := m.List.Mode
	oldCursor := m.List.Cursor

	m.List.Items = stacks

	// Restore filter query and mode
	m.List.Query = oldQuery
	m.List.Mode = oldMode

	// Re-apply filter to update filtered list
	if oldQuery != "" {
		m.List.ApplyFilter()
	} else {
		m.List.Filtered = stacks
	}

	// Compute stack-level error summary from snapshot so errors are visible
	// without expanding the stack (background check).
	// Reset maps first
	for k := range m.stackHasError {
		delete(m.stackHasError, k)
	}
	for k := range m.stackErrorText {
		delete(m.stackErrorText, k)
	}

	snap := docker.GetSnapshot()
	if snap != nil {
		// map service ID -> stack name
		svcToStack := make(map[string]string)
		for _, svc := range snap.Services {
			if svc.Spec.Labels != nil {
				if stackName, ok := svc.Spec.Labels["com.docker.stack.namespace"]; ok {
					svcToStack[svc.ID] = stackName
				}
			}
		}

		for _, t := range snap.Tasks {
			stackName := svcToStack[t.ServiceID]
			if stackName == "" {
				continue
			}
			if string(t.DesiredState) == "running" && t.Status.Err != "" {
				m.stackHasError[stackName] = true
				if m.stackErrorText[stackName] == "" {
					m.stackErrorText[stackName] = t.Status.Err
				}
			}
		}
	}

	// Auto-sort by error if errors exist and user hasn't manually set sort
	if !m.userSetSort {
		hasAnyError := false
		for _, hasErr := range m.stackHasError {
			if hasErr {
				hasAnyError = true
				break
			}
		}
		if hasAnyError {
			m.sortField = SortByError
			m.sortAscending = true
		}
	}

	// Reapply sorting to maintain sort order after data refresh
	m.applySorting()

	// Restore cursor position if still valid
	if oldCursor < len(m.List.Filtered) {
		m.List.Cursor = oldCursor
	} else if len(m.List.Filtered) > 0 {
		m.List.Cursor = len(m.List.Filtered) - 1
	} else {
		m.List.Cursor = 0
	}

	// If cursor is at 0 (initial state), ensure YOffset is also 0
	if m.List.Cursor == 0 {
		m.List.Viewport.YOffset = 0
	}

	m.setRenderItem()

	// Note: We don't call SetContent here because the View() method uses
	// VisibleContent() to render only the visible portion. Calling SetContent
	// with View() would cause conflicting YOffset adjustments.
	if m.ready {
		l().Info("StacksView.setStacks: Content ready for rendering")
	} else {
		l().Warn("StacksView.setStacks: View not ready yet")
	}
}

// After loading stacks, set RenderItem dynamically with correct column width
func (m *Model) setRenderItem() {
	defer func() {
		if r := recover(); r != nil {
			l().Errorf("panic in Stacks.setRenderItem: %v", r)
			l().Errorf("%s", debug.Stack())
		}
	}()
	// Build RenderItem to match the header layout used by the view (4 columns: STACK, SERVICES, TASKS, ERROR)
	itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("117"))

	m.List.RenderItem = func(s docker.StackEntry, selected bool, _ int) string {
		// Compute column widths similar to View()
		width := m.List.Viewport.Width
		if width <= 0 {
			width = m.width
		}
		if width <= 0 {
			width = 80
		}
		contentWidth := width
		// STACK: 25%, SERVICES: 10%, TASKS: 10%, ERROR: 55% (remainder)
		colWidths := make([]int, 4)
		colWidths[0] = (contentWidth * 25) / 100
		colWidths[1] = (contentWidth * 10) / 100
		colWidths[2] = (contentWidth * 10) / 100
		colWidths[3] = contentWidth - colWidths[0] - colWidths[1] - colWidths[2]

		// Update cached width
		m.width = width

		// Prepare stack name
		nameMax := colWidths[0] - 2
		if nameMax < 0 {
			nameMax = 0
		}
		name := s.Name
		if len(name) > nameMax {
			if nameMax > 3 {
				name = name[:nameMax-3] + "..."
			} else {
				name = name[:nameMax]
			}
		}

		svcStr := fmt.Sprintf("%d", s.ServiceCount)
		if len(svcStr) > colWidths[1] {
			svcStr = svcStr[:colWidths[1]]
		}

		nodeStr := fmt.Sprintf("%d", s.NodeCount)
		if len(nodeStr) > colWidths[2] {
			nodeStr = nodeStr[:colWidths[2]]
		}

		// Determine if the stack has an error: any task where desired is running and Error != ""
		stackHasError := false
		stackErrorText := ""
		if tasks, ok := m.stackTasks[s.Name]; ok {
			for _, t := range tasks {
				if strings.ToLower(t.DesiredState) == "running" && t.Error != "" {
					stackHasError = true
					stackErrorText = t.Error
					break
				}
			}
		}

		// For selected row, apply horizontal scroll to error text
		errorDisplayText := stackErrorText
		if selected && len(stackErrorText) > colWidths[3] {
			errorDisplayText = formatErrorWithScroll(stackErrorText, m.errorScrollOffset, colWidths[3])
		} else if len(stackErrorText) > colWidths[3] {
			errorDisplayText = truncateWithEllipsis(stackErrorText, colWidths[3])
		}

		// Render all columns in one format string using precision to truncate if needed
		if selected {
			selBg := lipgloss.Color("63")
			selStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(selBg).Bold(true)
			line := selStyle.Render(fmt.Sprintf(" %-*.*s%-*.*s%-*.*s%-*.*s",
				colWidths[0]-1, colWidths[0]-1, name,
				colWidths[1], colWidths[1], svcStr,
				colWidths[2], colWidths[2], nodeStr,
				colWidths[3], colWidths[3], errorDisplayText,
			))

			// If selected and expanded, render tasks below
			if m.expandedStacks[s.Name] {
				tasks := m.stackTasks[s.Name]
				taskHeaderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
				taskHeader := fmt.Sprintf("   %-22s  %-12s  %-13s  %-40s  %s",
					"NAME", "NODE", "DESIRED STATE", "CURRENT STATE", "ERROR")
				line += "\n" + taskHeaderStyle.Render(taskHeader)
				if len(tasks) > 0 {
					for ti, task := range tasks {
						taskName := truncateWithEllipsis(task.Name, 22)
						taskNode := truncateWithEllipsis(task.NodeName, 12)
						taskDesired := truncateWithEllipsis(task.DesiredState, 13)
						taskCurrent := truncateWithEllipsis(task.CurrentState, 40)
						taskErr := truncateWithEllipsis(task.Error, 30)
						taskSelected := m.selectedTaskIndex == ti
						if taskSelected {
							taskSelStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(lipgloss.Color("24")).Bold(true)
							line += "\n" + taskSelStyle.Render(fmt.Sprintf("   %-22s  %-12s  %-13s  %-40s  %s", taskName, taskNode, taskDesired, taskCurrent, taskErr))
						} else {
							taskStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
							line += "\n" + taskStyle.Render(fmt.Sprintf("   %-22s  %-12s  %-13s  %-40s  %s", taskName, taskNode, taskDesired, taskCurrent, taskErr))
						}
					}
				} else {
					line += "\n" + selStyle.Render("   (no tasks)")
				}
			}
			return line
		}

		// Non-selected: color entire row red if there's an error (like configs view)
		var baseStyle = itemStyle
		if stackHasError {
			baseStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
		}

		line := baseStyle.Render(fmt.Sprintf(" %-*.*s%-*.*s%-*.*s%-*.*s",
			colWidths[0]-1, colWidths[0]-1, name,
			colWidths[1], colWidths[1], svcStr,
			colWidths[2], colWidths[2], nodeStr,
			colWidths[3], colWidths[3], errorDisplayText,
		))

		if m.expandedStacks[s.Name] {
			tasks := m.stackTasks[s.Name]
			taskHeaderStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Italic(true)
			taskHeader := fmt.Sprintf("   %-22s  %-12s  %-13s  %-40s  %s",
				"NAME", "NODE", "DESIRED STATE", "CURRENT STATE", "ERROR")
			line += "\n" + taskHeaderStyle.Render(taskHeader)
			if len(tasks) > 0 {
				for _, task := range tasks {
					taskName := truncateWithEllipsis(task.Name, 22)
					taskNode := truncateWithEllipsis(task.NodeName, 12)
					taskDesired := truncateWithEllipsis(task.DesiredState, 13)
					taskCurrent := truncateWithEllipsis(task.CurrentState, 40)
					taskErr := truncateWithEllipsis(task.Error, 30)
					taskStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("7"))
					line += "\n" + taskStyle.Render(fmt.Sprintf("   %-22s  %-12s  %-13s  %-40s  %s", taskName, taskNode, taskDesired, taskCurrent, taskErr))
				}
			} else {
				line += "\n" + baseStyle.Render("   (no tasks)")
			}
		}
		return line
	}
}

// formatErrorWithScroll formats error text with horizontal scroll offset (like services view)
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

// truncateWithEllipsis matches the services view truncation semantics
func truncateWithEllipsis(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if maxWidth <= 1 {
		return "…"
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	ell := "…"
	ellW := lipgloss.Width(ell)
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

// GetStacksHelpContent returns categorized help for the stacks view
func GetStacksHelpContent() []helpview.HelpCategory {
	return []helpview.HelpCategory{
		{
			Title: "General",
			Items: []helpview.HelpItem{
				{Keys: "<i/enter>", Description: "Show services for Stack"},
				{Keys: "<p>", Description: "Show tasks for Stack"},
				{Keys: "<ctrl+d>", Description: "Delete stack"},
				{Keys: "</>", Description: "Filter"},
			},
		},
		{
			Title: "View",
			Items: []helpview.HelpItem{
				{Keys: "<shift+s>", Description: "Order by Stack name"},
				{Keys: "<shift+e>", Description: "Order by Services"},
				{Keys: "<shift+t>", Description: "Order by Tasks"},
				{Keys: "<shift+r>", Description: "Order by Error"},
			},
		},
		{
			Title: "Navigation",
			Items: []helpview.HelpItem{
				{Keys: "<↑/↓>", Description: "Navigate"},
				{Keys: "<pgup>", Description: "Page up"},
				{Keys: "<pgdown>", Description: "Page down"},
				{Keys: "<q>", Description: "Quit"},
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
	cursorStack := ""
	if m.List.Cursor < len(m.List.Filtered) {
		cursorStack = m.List.Filtered[m.List.Cursor].Name
	}

	// Sort the filtered list
	switch m.sortField {
	case SortByName:
		sort.Slice(m.List.Filtered, func(i, j int) bool {
			if m.sortAscending {
				return m.List.Filtered[i].Name < m.List.Filtered[j].Name
			}
			return m.List.Filtered[i].Name > m.List.Filtered[j].Name
		})
	case SortByServices:
		sort.Slice(m.List.Filtered, func(i, j int) bool {
			if m.sortAscending {
				return m.List.Filtered[i].ServiceCount < m.List.Filtered[j].ServiceCount
			}
			return m.List.Filtered[i].ServiceCount > m.List.Filtered[j].ServiceCount
		})
	case SortByTasks:
		sort.Slice(m.List.Filtered, func(i, j int) bool {
			if m.sortAscending {
				return m.List.Filtered[i].NodeCount < m.List.Filtered[j].NodeCount
			}
			return m.List.Filtered[i].NodeCount > m.List.Filtered[j].NodeCount
		})
	case SortByError:
		sort.Slice(m.List.Filtered, func(i, j int) bool {
			iHas := m.stackHasError[m.List.Filtered[i].Name]
			jHas := m.stackHasError[m.List.Filtered[j].Name]
			if iHas == jHas {
				// fallback to error text then name
				iText := m.stackErrorText[m.List.Filtered[i].Name]
				jText := m.stackErrorText[m.List.Filtered[j].Name]
				if iText == jText {
					if m.sortAscending {
						return m.List.Filtered[i].Name < m.List.Filtered[j].Name
					}
					return m.List.Filtered[i].Name > m.List.Filtered[j].Name
				}
				if m.sortAscending {
					return iText < jText
				}
				return iText > jText
			}
			// Place stacks with errors first when ascending
			if m.sortAscending {
				return iHas && !jHas
			}
			return !iHas && jHas
		})
	}

	// Restore cursor position to previously selected item
	if cursorStack != "" {
		for i, s := range m.List.Filtered {
			if s.Name == cursorStack {
				m.List.Cursor = i
				return
			}
		}
	}

	// If previous item not found, reset cursor to 0
	m.List.Cursor = 0
	m.List.Viewport.GotoTop()
}
