package servicesview

import (
	"context"
	"fmt"
	"swarmcli/docker"
	"swarmcli/views/confirmdialog"
	inspectview "swarmcli/views/inspect"
	loadingview "swarmcli/views/loading"
	logsview "swarmcli/views/logs"
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
			m.loading.SetVisible(true)
			m.loadingViewMessage(entry.ServiceName)
			l().Debugln("Starting restartServiceWithProgressCmd for", entry.ServiceName)

			// create new channel for this operation
			m.msgCh = make(chan tea.Msg)

			return m, tea.Batch(
				restartServiceWithProgressCmd(entry.ServiceName, m.msgCh),
				m.listenForMessages(),
			)
		}
		return m, nil

	case serviceProgressMsg:
		l().Debugf("[UI] Received progress: %d/%d\n", msg.Progress.Replaced, msg.Progress.Total)

		m.loadingViewMessage(fmt.Sprintf(
			"Progress: %d/%d tasks replaced...",
			msg.Progress.Replaced, msg.Progress.Total,
		))

		if msg.Progress.Replaced == msg.Progress.Total && msg.Progress.Total > 0 {
			l().Debugln("[UI] Restart finished")
			m.loading.SetVisible(false)
			return m, tea.Batch(
				refreshServicesCmd(m.nodeID, m.stackName, m.filterType),
			)
		}

		return m, m.listenForMessages()

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
		case "l":
			if m.cursor < len(m.entries) {
				entry := m.entries[m.cursor]
				return m, func() tea.Msg {
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
