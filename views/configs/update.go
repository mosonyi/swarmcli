package configsview

import (
	"fmt"
	"swarmcli/views/confirmdialog"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		l().Debugf("Window resized: width=%d, height=%d", msg.Width, msg.Height)
		m.list.SetSize(msg.Width, msg.Height-3)
		return nil

	case configsLoadedMsg:
		l().Infof("Loaded %d configs", len(msg))
		items := make([]list.Item, len(msg))
		for i, cfg := range msg {
			items[i] = configItemFromSwarm(cfg.Config)
		}
		m.configs = msg
		m.list.SetItems(items)
		m.state = stateReady
		return nil

	case configRotatedMsg:
		l().Infof("Config rotated: %s → %s", msg.Old.Config.Spec.Name, msg.New.Config.Spec.Name)
		return tea.Printf("Rotated %s → %s", msg.Old.Config.Spec.Name, msg.New.Config.Spec.Name)

	case configDeletedMsg:
		l().Infof("Config deleted successfully: %s", msg.Name)
		return loadConfigsCmd()

	case editConfigMsg:
		cfg := m.selectedConfig()
		l().Infof("Editing config: %s", cfg)
		return editConfigInEditorCmd(cfg)

	case editConfigDoneMsg:
		oldName := msg.OldConfig.Config.Spec.Name

		if !msg.Changed {
			l().Debugf("Edit finished: no changes detected for %s", oldName)
			return tea.Printf("No changes made to config: %s", oldName)
		}

		newName := msg.NewConfig.Config.Spec.Name

		l().Infof("Edit finished: config changed, inserting new version %s", newName)

		m.addConfig(msg.NewConfig)
		m.pendingAction = "rotate"
		m.configToRotateFrom = &msg.OldConfig
		m.configToRotateInto = &msg.NewConfig

		m.confirmDialog = m.confirmDialog.Show(
			fmt.Sprintf("Rotate from %s → %s now?", oldName, newName),
		)

		return tea.Printf("Config %s edited and queued for rotation", newName)

	case editConfigErrorMsg:
		l().Errorf("Error editing config: %v", msg.err)
		m.state = stateError
		m.err = msg.err
		return tea.Printf("Error editing config: %v", msg.err)

	case errorMsg:
		l().Errorf("Unhandled error: %v", msg)
		m.state = stateError
		m.err = msg
		return nil

	case confirmdialog.ResultMsg:
		l().Debugf("Confirm dialog result: confirmed=%v (pendingAction=%s)", msg.Confirmed, m.pendingAction)
		defer func() {
			m.pendingAction = ""
			m.confirmDialog.Visible = false
			m.configToRotateFrom = nil
			m.configToRotateInto = nil
			m.configToDelete = nil
		}()

		if !msg.Confirmed {
			l().Info("Action cancelled by user")
			m.confirmDialog.Visible = false
			return nil
		}

		switch m.pendingAction {
		case "rotate":
			if m.configToRotateInto == nil {
				l().Warnln("Confirmed rotation but configToRotate is nil")
				return nil
			}
			l().Infof("Confirmed rotation for %s", m.configToRotateInto.Config.Spec.Name)
			return rotateConfigCmd(m.configToRotateFrom, m.configToRotateInto)

		case "delete":
			if m.configToDelete == nil {
				l().Warnln("Confirmed delete but configToDelete is nil")
				return nil
			}
			name := m.configToDelete.Config.Spec.Name
			l().Infof("Confirmed deletion for config %s", name)
			return deleteConfigCmd(name)
		}

		return nil

	case tea.KeyMsg:
		if m.confirmDialog.Visible {
			l().Debugf("Key input routed to confirm dialog: %q", msg.String())
			cmd := m.confirmDialog.Update(msg)
			return cmd
		}

		switch msg.String() {
		case "r":
			cfgName := m.selectedConfig()
			if cfgName == "" {
				l().Warn("Rotate key pressed but no config selected")
				return nil
			}

			cfg, err := m.findConfigByName(cfgName)
			if err != nil {
				l().Errorf("Failed to find config %q for rotation: %v", cfgName, err)
				return tea.Printf("Cannot rotate: %v", err)
			}

			l().Infof("Rotate key pressed for config: %s", cfgName)

			m.pendingAction = "rotate"
			m.configToRotateInto = cfg
			m.confirmDialog = m.confirmDialog.Show(fmt.Sprintf("Rotate config %s?", cfgName))

			return nil

		case "d":
			if len(m.list.Items()) == 0 {
				return nil
			}
			selected, ok := m.list.SelectedItem().(configItem)
			if !ok {
				return nil
			}

			m.pendingAction = "delete"
			m.confirmDialog.Visible = true
			m.confirmDialog.Message = fmt.Sprintf("Delete config %s?", selected.Name)
			m.configToRotateInto = nil // just to be explicit
			m.configToDelete, _ = m.findConfigByName(selected.Name)

			return nil

		case "e":
			cfg := m.selectedConfig()
			l().Infof("Edit key pressed for config: %s", cfg)
			return editConfigInEditorCmd(cfg)

		case "enter":
			cfg := m.selectedConfig()
			l().Infof("Inspect key pressed for config: %s", cfg)
			return inspectConfigCmd(cfg)
		}
	}

	switch m.state {
	case stateLoading:
		var cmd tea.Cmd
		return cmd

	case stateReady:
		var cmd tea.Cmd
		m.list, cmd = m.list.Update(msg)
		return cmd

	default:
		return nil
	}
}
