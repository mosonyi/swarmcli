package servicesview

import (
	"context"
	"fmt"
	"swarmcli/core/primitives/hash"
	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/confirmdialog"
	inspectview "swarmcli/views/inspect"
	logsview "swarmcli/views/logs"
	"swarmcli/views/view"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {

	case Msg:
		l().Infof("ServicesView: Received Msg with %d entries", len(msg.Entries))
		// Update the hash with new data
		var err error
		m.lastSnapshot, err = hash.Compute(msg.Entries)
		if err != nil {
			l().Errorf("ServicesView: Error computing hash: %v", err)
			return nil
		}
		m.SetContent(msg)
		m.Visible = true
		m.List.Viewport.SetContent(m.List.View())
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

	case tea.WindowSizeMsg:
		m.List.Viewport.Width = msg.Width
		m.List.Viewport.Height = msg.Height
		m.ready = true
		m.List.Viewport.SetContent(m.List.View())
		return nil
	case confirmdialog.ResultMsg:
		m.confirmDialog.Visible = false

		if msg.Confirmed && m.List.Cursor < len(m.List.Filtered) {
			entry := m.List.Filtered[m.List.Cursor]
			m.loading.SetVisible(true)
			m.loadingViewMessage(entry.ServiceName)
			l().Debugln("Starting restartServiceWithProgressCmd for", entry.ServiceName)

			// create new channel for this operation
			m.msgCh = make(chan tea.Msg)

			return tea.Batch(
				restartServiceWithProgressCmd(entry.ServiceName, m.msgCh),
				m.listenForMessages(),
			)
		}
		return nil

	case serviceProgressMsg:
		l().Debugf("[UI] Received progress: %d/%d\n", msg.Progress.Replaced, msg.Progress.Total)

		m.loadingViewMessage(fmt.Sprintf(
			"Progress: %d/%d tasks replaced...",
			msg.Progress.Replaced, msg.Progress.Total,
		))

		if msg.Progress.Replaced == msg.Progress.Total && msg.Progress.Total > 0 {
			l().Debugln("[UI] Restart finished")
			m.loading.SetVisible(false)
			return tea.Batch(
				refreshServicesCmd(m.nodeID, m.stackName, m.filterType),
			)
		}

		return m.listenForMessages()

	case tea.KeyMsg:
		if m.confirmDialog.Visible {
			return m.confirmDialog.Update(msg)
		}

		if m.loading.Visible() {
			return nil
		}

		// --- if in search mode, handle all keys via FilterableList ---
		if m.List.Mode == filterlist.ModeSearching {
			m.List.HandleKey(msg)
			return nil
		}

		// --- normal mode ---
		m.List.HandleKey(msg) // still handle up/down/pgup/pgdown

		switch msg.String() {
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
				m.confirmDialog.Visible = true
				m.confirmDialog.Message = fmt.Sprintf("Restart service %q?", entry.ServiceName)
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
		m.List.Viewport.GotoTop()
	}
}

func (m *Model) setRenderItem() {
	// We'll partition the available width into equal columns (6 columns):
	// SERVICE | STACK | REPLICAS | STATUS | CREATED | UPDATED
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

	m.List.RenderItem = func(e docker.ServiceEntry, selected bool, _ int) string {
		// Recompute column widths on each render so items adapt to viewport resizes
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

		// Update cached widths so header aligns
		m.colService = colWidths[0]
		m.colStack = colWidths[1]

		// Prepare texts
		replicasText := fmt.Sprintf("%d/%d", e.ReplicasOnNode, e.ReplicasTotal)
		if e.ReplicasTotal == 0 {
			replicasText = "—"
		}

		// Truncate strings to their column width
		serviceName := truncateWithEllipsis(e.ServiceName, colWidths[0])
		stackName := truncateWithEllipsis(e.StackName, colWidths[1])
		statusText := truncateWithEllipsis(e.Status, colWidths[3])
		created := truncateWithEllipsis(formatRelativeTime(e.CreatedAt), colWidths[4])
		updated := truncateWithEllipsis(formatRelativeTime(e.UpdatedAt), colWidths[5])

		// Build each column text and apply coloring where appropriate
		// Service + Stack use base item style (white content) and reserve a leading space
		itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("15"))
		// Keep one leading space in first column so it aligns with header
		col0 := itemStyle.Render(fmt.Sprintf(" %-*s", colWidths[0]-1, serviceName))
		col1 := itemStyle.Render(fmt.Sprintf("%-*s", colWidths[1], stackName))

		// Replicas colored
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
		col2 := replicasStyle.Render(fmt.Sprintf("%-*s", colWidths[2], replicasText))

		// Status colored
		statusColor := getStatusColor(e.Status)
		statusStyle := lipgloss.NewStyle().Foreground(statusColor)
		col3 := statusStyle.Render(fmt.Sprintf("%-*s", colWidths[3], statusText))

		// Created/Updated use base style
		col4 := itemStyle.Render(fmt.Sprintf("%-*s", colWidths[4], created))
		col5 := itemStyle.Render(fmt.Sprintf("%-*s", colWidths[5], updated))

		line := col0 + col1 + col2 + col3 + col4 + col5

		if selected {
			selBg := lipgloss.Color("63")
			// Base selected style for text columns
			selBase := lipgloss.NewStyle().Foreground(lipgloss.Color("230")).Background(selBg).Bold(true)
			// Replicas selected style
			selRep := lipgloss.NewStyle().Foreground(replicasColor).Background(selBg).Bold(true)
			// Status selected style
			selStatus := lipgloss.NewStyle().Foreground(statusColor).Background(selBg).Bold(true)

			// Preserve leading space when selected as well
			col0 = selBase.Render(fmt.Sprintf(" %-*s", colWidths[0]-1, serviceName))
			col1 = selBase.Render(fmt.Sprintf("%-*s", colWidths[1], stackName))
			col2 = selRep.Render(fmt.Sprintf("%-*s", colWidths[2], replicasText))
			col3 = selStatus.Render(fmt.Sprintf("%-*s", colWidths[3], statusText))
			col4 = selBase.Render(fmt.Sprintf("%-*s", colWidths[4], created))
			col5 = selBase.Render(fmt.Sprintf("%-*s", colWidths[5], updated))
			line = col0 + col1 + col2 + col3 + col4 + col5
		}
		return line
	}
}

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
