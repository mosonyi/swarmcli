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
		if msg.Confirmed {
			entry := m.entries[m.cursor]
			m.confirmDialog.Visible = false
			m.loading.SetVisible(true)
			m.loadingViewMessage(entry.ServiceName)
			return m, restartServiceCmd(entry.ServiceName, m.filterType, m.nodeID, m.stackName)
		}
		m.confirmDialog.Visible = false

	case tea.KeyMsg:
		if m.confirmDialog.Visible {
			var cmd tea.Cmd
			m.confirmDialog, cmd = m.confirmDialog.Update(msg)
			return m, cmd
		}

		// If loading visible, ignore user input
		if m.loading.Visible() {
			return m, nil
		}

		switch msg.String() {
		case "q":
			m.Visible = false

		case "j", "down":
			if m.cursor < len(m.entries)-1 {
				m.cursor++
				m.viewport.SetContent(m.renderEntries())
			}

		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
				m.viewport.SetContent(m.renderEntries())
			}

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
		case "r":
			if m.cursor < len(m.entries) {
				entry := m.entries[m.cursor]
				m.confirmDialog.Visible = true
				m.confirmDialog.Message = fmt.Sprintf("Restart service %q?", entry.ServiceName)
			}
		}

	default:
		// Allow spinner updates while loading
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
