package configsview

import (
	"fmt"
	"swarmcli/views/confirmdialog"
	"swarmcli/views/view"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (view.View, tea.Cmd) {

	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.list.SetSize(msg.Width, msg.Height-3)
		return m, nil

	case configsLoadedMsg:
		items := make([]list.Item, len(msg))
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

	case editConfigMsg:
		// Launch editor via ExecProcess, suspending TUI automatically
		return m, editConfigInEditorCmd(m.selectedConfig())

	case editConfigDoneMsg:
		if msg.Changed {
			// Insert the new version into the list
			m.list.InsertItem(0, configItemFromSwarm(msg.Config.Config))

			// Ask user if they want to rotate the new version
			m.pendingAction = "rotate"
			m.confirmDialog.Visible = true
			m.confirmDialog.Message = fmt.Sprintf("Rotate config %s now?", msg.Config.Config.Spec.Name)
		}
		return m, tea.Printf("Edited config: %s", msg.Name)

	case editConfigErrorMsg:
		m.state = stateError
		m.err = msg.err
		return m, tea.Printf("Error editing config: %v", msg.err)

	case errorMsg:
		m.state = stateError
		m.err = msg
		return m, nil

	case confirmdialog.ResultMsg:
		l().Debugln("Confirm dialog result")
		if msg.Confirmed {
			l().Debugln("Confirm dialog confirmed")
			cfg := m.selectedConfig()
			switch m.pendingAction {
			case "rotate":
				m.pendingAction = ""
				m.confirmDialog.Visible = false
				return m, rotateConfigCmd(cfg)
			}
		} else {
			m.pendingAction = ""
			m.confirmDialog.Visible = false
		}

	case tea.KeyMsg:
		if m.confirmDialog.Visible {
			var cmd tea.Cmd
			m.confirmDialog, cmd = m.confirmDialog.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "r":
			cfgName := m.selectedConfig()
			if cfgName == "" {
				return m, nil
			}
			// Show confirm dialog
			m.pendingAction = "rotate"
			m.confirmDialog.Visible = true
			m.confirmDialog.Message = fmt.Sprintf("Rotate config %s?", cfgName)
			return m, nil
		case "e":
			return m, editConfigInEditorCmd(m.selectedConfig())
		case "enter":
			return m, inspectConfigCmd(m.selectedConfig())
		}
	}

	switch m.state {
	case stateLoading:
		var cmd tea.Cmd
		// m.loadingView, cmd = m.loadingView.Update(msg)
		return m, cmd
	case stateReady:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd
	default:
		return m, nil
	}
}
