package nodeservicesview

import (
	"context"
	"fmt"
	"swarmcli/docker"
	"swarmcli/views/confirmdialog"
	inspectview "swarmcli/views/inspect"
	loadingview "swarmcli/views/loading"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (view.View, tea.Cmd) {
	switch msg := msg.(type) {

	case Msg:
		m.SetContent(msg)
		m.Visible = true
		m.viewport.SetContent(m.renderEntries())
		return m, nil

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		if !m.ready {
			m.ready = true
			m.viewport.SetContent(m.renderEntries())
		}
		return m, nil

	case confirmdialog.ResultMsg:
		m.confirmDialog.Visible = false

		if msg.Confirmed && m.cursor < len(m.entries) {
			entry := m.entries[m.cursor]

			// Show loading spinner
			m.loading.SetVisible(true)
			m.loadingViewMessage(entry.ServiceName)

			// Restart asynchronously; spinner will animate while waiting
			return m, restartServiceCmd(entry.ServiceName)
		}
		return m, nil

	case serviceRestartProgressMsg:
		m.loadingViewMessage(fmt.Sprintf(
			"Restarting %s: %d/%d tasks replaced...",
			msg.ServiceName, msg.Running, msg.Replicas,
		))
		return m, nil

	case serviceRestartedMsg:
		m.loading.SetVisible(false)
		if msg.Err != nil {
			fmt.Printf("Failed to restart service %q: %v\n", msg.ServiceName, msg.Err)
		} else {
			return m, refreshServicesCmd(m.nodeID, m.stackName, m.filterType)
		}
		return m, nil

	case tea.KeyMsg:
		if m.confirmDialog.Visible {
			var cmd tea.Cmd
			m.confirmDialog, cmd = m.confirmDialog.Update(msg)
			return m, cmd
		}

		// 2. Ignore keys if loading visible ---
		if m.loading.Visible() {
			return m, nil
		}

		switch msg.String() {
		case "q":
			m.Visible = false
			return m, nil

		case "j", "down":
			if m.cursor < len(m.entries)-1 {
				m.cursor++
				m.viewport.SetContent(m.renderEntries())
			}
			return m, nil

		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				m.viewport.SetContent(m.renderEntries())
			}
			return m, nil

		case "i":
			if m.cursor < len(m.entries) {
				entry := m.entries[m.cursor]
				return m, func() tea.Msg {
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
			return m, nil

		case "r":
			if m.cursor < len(m.entries) {
				entry := m.entries[m.cursor]
				m.confirmDialog.Visible = true
				m.confirmDialog.Message = fmt.Sprintf("Restart service %q?", entry.ServiceName)
			}
			return m, nil
		}

	// --- Allow spinner updates while loading ---
	default:
		// Forward messages to loading view if active
		if m.loading.Visible() {
			var cmd tea.Cmd
			var v view.View
			v, cmd = m.loading.Update(msg)
			m.loading = v.(loadingview.Model)
			return m, cmd
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return m, cmd
}
