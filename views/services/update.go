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

	// Ensure total width fits viewport
	total := maxService + maxStack + replicaWidth + 2*gap
	if total > width {
		overflow := total - width
		if maxStack > maxService {
			maxStack -= overflow
			if maxStack < 5 {
				maxStack = 5
			}
		} else {
			maxService -= overflow
			if maxService < 5 {
				maxService = 5
			}
		}
	}

	m.List.RenderItem = func(e docker.ServiceEntry, selected bool, _ int) string {
		replicas := fmt.Sprintf("%d/%d", e.ReplicasOnNode, e.ReplicasTotal)
		switch {
		case e.ReplicasTotal == 0:
			replicas = lipgloss.NewStyle().Foreground(lipgloss.Color("8")).Render("—")
		case e.ReplicasOnNode == 0:
			replicas = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(replicas)
		case e.ReplicasOnNode < e.ReplicasTotal:
			replicas = lipgloss.NewStyle().Foreground(lipgloss.Color("11")).Render(replicas)
		default:
			replicas = lipgloss.NewStyle().Foreground(lipgloss.Color("10")).Render(replicas)
		}

		serviceName := truncateWithEllipsis(e.ServiceName, maxService)
		stackName := truncateWithEllipsis(e.StackName, maxStack)

		line := fmt.Sprintf(
			"%-*s        %-*s        %*s",
			maxService, serviceName,
			maxStack, stackName,
			replicaWidth, replicas,
		)

		if selected {
			line = ui.CursorStyle.Render(line)
		} else {
			// Apply light blue color to the entire line (preserving replica colors)
			// Since replicas already has color styling, we need to keep it separate
			itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
			serviceAndStack := fmt.Sprintf("%-*s        %-*s        ", maxService, serviceName, maxStack, stackName)
			line = itemStyle.Render(serviceAndStack) + replicas
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
