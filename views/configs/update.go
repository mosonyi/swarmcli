package configsview

import (
	"fmt"
	"swarmcli/core/primitives/hash"
	"swarmcli/ui"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/confirmdialog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.configsList.Viewport.Width = msg.Width
		m.configsList.Viewport.Height = msg.Height - 3
		return nil

	case configsLoadedMsg:
		l().Infof("ConfigsView: Received configsLoadedMsg with %d configs", len(msg))
		// Update the hash with new data
		var err error
		m.lastSnapshot, err = hash.Compute(msg)
		if err != nil {
			l().Errorf("ConfigsView: Error computing hash: %v", err)
		}

		// Preserve current cursor position
		oldCursor := m.configsList.Cursor

		m.configs = msg
		items := make([]configItem, len(msg))
		for i, cfg := range msg {
			items[i] = configItemFromSwarm(cfg.Config)
		}
		m.configsList.Items = items
		m.setRenderItem()
		m.configsList.ApplyFilter()

		// Restore cursor position, but ensure it's within bounds
		if oldCursor < len(m.configsList.Filtered) {
			m.configsList.Cursor = oldCursor
		} else if len(m.configsList.Filtered) > 0 {
			m.configsList.Cursor = len(m.configsList.Filtered) - 1
		} else {
			m.configsList.Cursor = 0
		}

		m.state = stateReady
		l().Info("ConfigsView: Config list updated")
		return nil

	case TickMsg:
		l().Infof("ConfigsView: Received TickMsg, state=%v", m.state)
		// Check for changes (this will return either configsLoadedMsg or the next TickMsg)
		if m.state == stateReady {
			return CheckConfigsCmd(m.lastSnapshot)
		}
		// Continue polling even if not ready
		return m.tickCmd()

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
		m.setRenderItem()
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
			return m.confirmDialog.Update(msg)
		}

		// --- if in search mode, handle all keys via FilterableList ---
		if m.configsList.Mode == filterlist.ModeSearching {
			m.configsList.HandleKey(msg)
			return nil
		}

		// --- normal mode ---
		m.configsList.HandleKey(msg) // still handle up/down/pgup/pgdown

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
			if len(m.configsList.Filtered) == 0 {
				return nil
			}
			cfgName := m.selectedConfig()
			cfg, _ := m.findConfigByName(cfgName)
			m.pendingAction = "delete"
			m.configToDelete = cfg
			m.confirmDialog = m.confirmDialog.Show(fmt.Sprintf("Delete config %s?", cfgName))
			return nil

		case "e":
			cfg := m.selectedConfig()
			l().Infof("Edit key pressed for config: %s", cfg)
			return editConfigInEditorCmd(m.selectedConfig())
		case "i":
			cfg := m.selectedConfig()
			l().Infof("Inspect key pressed for config: %s", cfg)
			return inspectConfigCmd(m.selectedConfig())
		case "enter":
			cfg := m.selectedConfig()
			l().Infof("Inspect key pressed for config: %s", cfg)
			return inspectRawConfigCmd(m.selectedConfig())
		}
	}

	// --- State-based Update ---
	switch m.state {
	case stateReady:
		// nothing extra for now; viewport already handled
		return nil
	default:
		return nil
	}
}

func (m *Model) setRenderItem() {
	// Compute max width per column
	nameCol := 0
	idCol := 0
	for _, cfg := range m.configsList.Items {
		if len(cfg.Name) > nameCol {
			nameCol = len(cfg.Name)
		}
		if len(cfg.ID) > idCol {
			idCol = len(cfg.ID)
		}
	}

	// Assign to the list
	itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
	m.configsList.RenderItem = func(cfg configItem, selected bool, _ int) string {
		line := fmt.Sprintf("%-*s        %-*s", nameCol, cfg.Name, idCol, cfg.ID)
		if selected {
			return ui.CursorStyle.Render(line)
		}
		return itemStyle.Render(line)
	}
}
