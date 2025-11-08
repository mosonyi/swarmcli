package configsview

import (
	"fmt"
	"swarmcli/views/confirmdialog"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (view.View, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-3)
		m.confirmDialog.Width = msg.Width
		m.confirmDialog.Height = msg.Height
		return m, nil

	// ---- Async results ----
	case configsLoadedMsg:
		items := make([]tea.Item, len(msg))
		for i, cfg := range msg {
			items[i] = configItemFromSwarm(cfg.Config)
		}
		m.list.SetItems(items)
		m.state = stateReady
		return m, nil

	case configUpdatedMsg:
		m.list.InsertItem(0, configItemFromSwarm(msg.New.Config))
		return m, tea.Printf("Created new config version: %s", msg.New.Config.Spec.Name)

	case configRotatedMsg:
		return m, tea.Printf("Rotated %s → %s", msg.Old.Config.Spec.Name, msg.New.Config.Spec.Name)

	case errorMsg:
		m.state = stateError
		m.err = msg
		return m, nil

	// ---- Confirmation handling ----
	case tea.KeyMsg:
		if m.confirmDialog.Visible {
			var cmd tea.Cmd
			m.confirmDialog, cmd = m.confirmDialog.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "q", "esc":
			return nil, view.SwitchBack()
		case "r":
			cfg := m.selectedConfig()
			if cfg == "" {
				return m, nil
			}
			m.pendingAction = "rotate"
			m.confirmDialog.Visible = true
			m.confirmDialog.Message = fmt.Sprintf("Rotate config %q across all services?", cfg)
			return m, nil
		case "e":
			return m, editConfigCmd(m.selectedConfig())
		case "enter":
			return m, inspectConfigCmd(m.selectedConfig())
		}

	case confirmdialog.ResultMsg:
		if !msg.Confirmed {
			m.pendingAction = ""
			m.confirmDialog.Visible = false
			return m, nil
		}

		cfg := m.selectedConfig()
		switch m.pendingAction {
		case "rotate":
			m.pendingAction = ""
			m.confirmDialog.Visible = false
			return m, rotateConfigCmd(cfg)
		}
	}

	// ---- State-driven updates ----
	switch m.state {
	case stateLoading:
		var cmd tea.Cmd
		m.loadingView, cmd = m.loadingView.Update(msg)
		return m, cmd
	case stateReady:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}
