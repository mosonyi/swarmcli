package servicesview

import (
	"context"
	"fmt"
	"swarmcli/docker"
	"swarmcli/views/confirmdialog"
	inspectview "swarmcli/views/inspect"
	logsview "swarmcli/views/logs"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {

	case Msg:
		m.SetContent(msg)
		m.Visible = true
		m.viewport.SetContent(m.renderEntries())
		return nil

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		if !m.ready {
			m.ready = true
			m.viewport.SetContent(m.renderEntries())
		}
		return nil

	case confirmdialog.ResultMsg:
		m.confirmDialog.Visible = false

		if msg.Confirmed && m.cursor < len(m.entries) {
			entry := m.entries[m.cursor]
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
			cmd := m.confirmDialog.Update(msg)
			return cmd
		}

		// 2. Ignore keys if loading visible ---
		if m.loading.Visible() {
			return nil
		}

		switch msg.String() {
		case "q":
			m.Visible = false
			return nil

		case "j", "down":
			if m.cursor < len(m.entries)-1 {
				m.cursor++
				m.viewport.SetContent(m.renderEntries())
			}
			return nil

		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				m.viewport.SetContent(m.renderEntries())
			}
			return nil

		case "i":
			if m.cursor < len(m.entries) {
				entry := m.entries[m.cursor]
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
			return nil

		case "r":
			if m.cursor < len(m.entries) {
				entry := m.entries[m.cursor]
				m.confirmDialog.Visible = true
				m.confirmDialog.Message = fmt.Sprintf("Restart service %q?", entry.ServiceName)
			}
			return nil
		case "l":
			if m.cursor < len(m.entries) {
				entry := m.entries[m.cursor]
				return func() tea.Msg {
					//content, err := docker.Inspect(context.Background(), docker.InspectService, entry.ServiceID)
					//if err != nil {
					//	content = fmt.Sprintf("Error inspecting service %q: %v", entry.ServiceName, err)
					//}
					return view.NavigateToMsg{
						Payload:  entry,
						ViewName: logsview.ViewName,
						//Payload: map[string]interface{}{
						//	"title": fmt.Sprintf("Service: %s", entry.ServiceName),
						//	"json":  content,
						//},
					}
				}
			}
			return nil
		}

	// --- Allow spinner updates while loading ---
	default:
		// Forward messages to loading view if active
		if m.loading.Visible() {
			cmd := m.loading.Update(msg)
			return cmd
		}
	}

	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	return cmd
}
