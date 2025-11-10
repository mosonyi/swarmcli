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
		m.configs = msg
		m.list.SetItems(items)
		m.state = stateReady
		return m, nil

	case configRotatedMsg:
		l().Infof("Config rotated: %s → %s", msg.Old.Config.Spec.Name, msg.New.Config.Spec.Name)
		return m, tea.Printf("Rotated %s → %s", msg.Old.Config.Spec.Name, msg.New.Config.Spec.Name)

	case editConfigMsg:
		cfg := m.selectedConfig()
		l().Infof("Editing config: %s", cfg)
		return m, editConfigInEditorCmd(cfg)

	case editConfigDoneMsg:
		oldName := msg.OldConfig.Config.Spec.Name

		if !msg.Changed {
			l().Debugf("Edit finished: no changes detected for %s", oldName)
			return m, tea.Printf("No changes made to config: %s", oldName)
		}

		newName := msg.NewConfig.Config.Spec.Name

		l().Infof("Edit finished: config changed, inserting new version %s", newName)

		m.list.InsertItem(0, configItemFromSwarm(msg.NewConfig.Config))
		m.configs = append(m.configs, msg.NewConfig)
		m.pendingAction = "rotate"
		m.configToRotateFrom = &msg.OldConfig
		m.configToRotateInto = &msg.NewConfig

		m.confirmDialog.Show(fmt.Sprintf("Rotate config %s now?", newName))

		return m, tea.Printf("Config %s edited and queued for rotation", newName)

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
		defer func() {
			m.pendingAction = ""
			m.configToRotateInto = nil
			m.confirmDialog = m.confirmDialog.Hide()
		}()

		if msg.Confirmed {
			if m.configToRotateInto == nil {
				l().Warnln("Confirmed in dialog, but configToRotateInto is nil. This is a bug!")
				return m, nil
			}

			l().Infof("Confirmed rotation for %s", m.configToRotateInto.Config.Spec.Name)
			return m, rotateConfigCmd(m.configToRotateFrom, m.configToRotateInto)
		}

		l().Info("Rotation cancelled by user")
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
			cfgName := m.selectedConfig()
			if cfgName == "" {
				l().Warn("Rotate key pressed but no config selected")
				return m, nil
			}

			cfg, err := m.findConfigByName(cfgName)
			if err != nil {
				l().Errorf("Failed to find config %q for rotation: %v", cfgName, err)
				return m, tea.Printf("Cannot rotate: %v", err)
			}

			l().Infof("Rotate key pressed for config: %s", cfgName)

			m.pendingAction = "rotate"
			m.configToRotateInto = cfg
			m.confirmDialog = m.confirmDialog.Show(fmt.Sprintf("Rotate config %s?", cfgName))

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
		var cmd tea.Cmd
		return m, cmd

	case stateReady:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return m, cmd

	default:
		return m, nil
	}
}
