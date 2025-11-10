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
		l().Debugf("Window resized: width=%d, height=%d", msg.Width, msg.Height)
		m.list.SetSize(msg.Width, msg.Height-3)
		return m, nil

	case configsLoadedMsg:
		l().Infof("Loaded %d configs", len(msg))
		items := make([]list.Item, len(msg))
		for i, cfg := range msg {
			items[i] = configItemFromSwarm(cfg.Config)
		}
		m.list.SetItems(items)
		m.state = stateReady
		return m, nil

	case configUpdatedMsg:
		l().Infof("Config updated: created new version %s", msg.New.Config.Spec.Name)
		m.list.InsertItem(0, configItemFromSwarm(msg.New.Config))
		return m, tea.Printf("Created new config version: %s", msg.New.Config.Spec.Name)

	case configRotatedMsg:
		l().Infof("Config rotated: %s → %s", msg.Old.Config.Spec.Name, msg.New.Config.Spec.Name)
		return m, tea.Printf("Rotated %s → %s", msg.Old.Config.Spec.Name, msg.New.Config.Spec.Name)

	case editConfigMsg:
		cfg := m.selectedConfig()
		l().Infof("Editing config: %s", cfg)
		return m, editConfigInEditorCmd(cfg)

	case editConfigDoneMsg:
		if msg.Changed {
			l().Infof("Edit finished: config changed, inserting new version %s", msg.Config.Config.Spec.Name)
			m.list.InsertItem(0, configItemFromSwarm(msg.Config.Config))

			m.pendingAction = "rotate"
			m.confirmDialog.Visible = true
			m.confirmDialog.Message = fmt.Sprintf("Rotate config %s now?", msg.Config.Config.Spec.Name)

			m.configToRotate = &msg.Config
		} else {
			l().Debugf("Edit finished: no changes detected for %s", msg.Name)
		}
		return m, tea.Printf("Edited config: %s", msg.Name)

	case editConfigErrorMsg:
		l().Errorf("Error editing config: %v", msg.err)
		m.state = stateError
		m.err = msg.err
		return m, tea.Printf("Error editing config: %v", msg.err)

	case errorMsg:
		l().Errorf("Unhandled error: %v", msg)
		m.state = stateError
		m.err = msg
		return m, nil

	case confirmdialog.ResultMsg:
		l().Debugf("Confirm dialog result: confirmed=%v (pendingAction=%s)", msg.Confirmed, m.pendingAction)
		if msg.Confirmed {
			if m.configToRotate == nil {
				l().Warnln("Confirmed in dialog, but config to rotate is nil. This is a bug!")
				m.pendingAction = ""
				m.confirmDialog.Visible = false
				return m, nil
			}
			l().Infof("Confirmed rotation for %s", m.configToRotate.Config.Spec.Name)
			m.pendingAction = ""
			m.confirmDialog.Visible = false
			cmd := rotateConfigCmd(m.configToRotate)
			m.configToRotate = nil
			return m, cmd
		}

		l().Info("Rotation cancelled by user")
		m.pendingAction = ""
		m.confirmDialog.Visible = false
		m.configToRotate = nil
		return m, nil

	case tea.KeyMsg:
		if m.confirmDialog.Visible {
			l().Debugf("Key input routed to confirm dialog: %q", msg.String())
			var cmd tea.Cmd
			m.confirmDialog, cmd = m.confirmDialog.Update(msg)
			return m, cmd
		}

		switch msg.String() {
		case "r":
			cfg := m.selectedConfig()
			if cfg == "" {
				l().Warn("Rotate key pressed but no config selected")
				return m, nil
			}
			l().Infof("Rotate key pressed for config: %s", cfg)
			m.pendingAction = "rotate"
			m.confirmDialog.Visible = true
			m.confirmDialog.Message = fmt.Sprintf("Rotate config %s?", cfg)
			return m, nil

		case "e":
			cfg := m.selectedConfig()
			l().Infof("Edit key pressed for config: %s", cfg)
			return m, editConfigInEditorCmd(cfg)

		case "enter":
			cfg := m.selectedConfig()
			l().Infof("Inspect key pressed for config: %s", cfg)
			return m, inspectConfigCmd(cfg)
		}
	}

	switch m.state {
	case stateLoading:
		//l().Debugln("State=Loading: skipping list updates")
		var cmd tea.Cmd
		// m.loadingView, cmd = m.loadingView.Update(msg)
		return m, cmd

	case stateReady:
		//l().Debugln("State=Ready: updating list")
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd

	default:
		//l().Warnf("Unhandled state: %v", m.state)
		return m, nil
	}
}
