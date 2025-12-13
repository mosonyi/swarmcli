package servicesview

import (
	"context"
	"fmt"
	"swarmcli/core/primitives/hash"
	"swarmcli/docker"
	"swarmcli/ui"
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
	const replicaWidth = 10
	const statusWidth = 12
	const createdWidth = 10
	const updatedWidth = 10
	const gap = 2

	width := m.List.Viewport.Width
	if width <= 0 {
		width = 80
	}

	// Compute the longest service and stack names in the filtered list
	maxService := len("SERVICE")
	maxStack := len("STACK")
	for _, e := range m.List.Filtered {
		if len(e.ServiceName) > maxService {
			maxService = len(e.ServiceName)
		}
		if len(e.StackName) > maxStack {
			maxStack = len(e.StackName)
		}
	}

	// Use DistributeColumns to ensure columns fill the viewport width.
	cols := []int{maxService, maxStack, replicaWidth, statusWidth, createdWidth, updatedWidth}
	// There are 5 gaps of 8 spaces in the formatted line
	adjusted := ui.DistributeColumns(width, 5, 8, cols, []int{0})
	maxService = adjusted[0]
	maxStack = adjusted[1]

	// Cache column widths on the model so the view header can align exactly
	m.colService = maxService
	m.colStack = maxStack

	m.List.RenderItem = func(e docker.ServiceEntry, selected bool, _ int) string {
		// Format plain text first for proper alignment
		replicasText := fmt.Sprintf("%d/%d", e.ReplicasOnNode, e.ReplicasTotal)
		if e.ReplicasTotal == 0 {
			replicasText = "—"
		}

		serviceName := truncateWithEllipsis(e.ServiceName, maxService)
		stackName := truncateWithEllipsis(e.StackName, maxStack)
		statusText := e.Status
		created := formatRelativeTime(e.CreatedAt)
		updated := formatRelativeTime(e.UpdatedAt)

		// Build the line with proper spacing
		line := fmt.Sprintf(
			"%-*s        %-*s        %-*s        %-*s        %-*s        %-*s",
			maxService, serviceName,
			maxStack, stackName,
			replicaWidth, replicasText,
			statusWidth, statusText,
			createdWidth, created,
			updatedWidth, updated,
		)

		if selected {
			line = ui.CursorStyle.Render(line)
		} else {
			// Apply colors using lipgloss styles on individual columns
			itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("117"))

			// Color replicas based on status
			var replicasColor lipgloss.Color
			switch {
			case e.ReplicasTotal == 0:
				replicasColor = lipgloss.Color("8") // gray
			case e.ReplicasOnNode == 0:
				replicasColor = lipgloss.Color("9") // red
			case e.ReplicasOnNode < e.ReplicasTotal:
				replicasColor = lipgloss.Color("11") // yellow
			default:
				replicasColor = lipgloss.Color("10") // green
			}
			replicasStyle := lipgloss.NewStyle().Foreground(replicasColor)

			// Color status
			statusColor := getStatusColor(e.Status)
			statusStyle := lipgloss.NewStyle().Foreground(statusColor)

			// Build colored line maintaining exact spacing
			line = itemStyle.Render(fmt.Sprintf("%-*s        %-*s        ", maxService, serviceName, maxStack, stackName)) +
				replicasStyle.Render(fmt.Sprintf("%-*s", replicaWidth, replicasText)) +
				itemStyle.Render("        ") +
				statusStyle.Render(fmt.Sprintf("%-*s", statusWidth, statusText)) +
				itemStyle.Render(fmt.Sprintf("        %-*s        %-*s", createdWidth, created, updatedWidth, updated))
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
