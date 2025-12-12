package configsview

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"swarmcli/core/primitives/hash"
	"swarmcli/docker"
	"swarmcli/ui"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/confirmdialog"
	view "swarmcli/views/view"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.configsList.Viewport.Width = msg.Width
		// msg.Height is already adjusted by the app to account for the
		// systeminfo header; avoid subtracting extra lines here.
		m.configsList.Viewport.Height = msg.Height
		return nil

	case configsLoadedMsg:
		l().Infof("ConfigsView: Received configsLoadedMsg with %d configs", len(msg))
		// Update the hash with new data using stable fields only
		type stableConfig struct {
			ID      string
			Version uint64
			Name    string
		}
		stableConfigs := make([]stableConfig, len(msg))
		for i, c := range msg {
			stableConfigs[i] = stableConfig{
				ID:      c.Config.ID,
				Version: c.Config.Version.Index,
			}
		}
		var err error
		m.lastSnapshot, err = hash.Compute(stableConfigs)
		if err != nil {
			l().Errorf("ConfigsView: Error computing hash: %v", err)
		}

		m.configs = msg
		items := make([]configItem, len(msg))
		ctx := context.Background()
		for i, cfg := range msg {
			items[i] = configItemFromSwarm(ctx, cfg.Config)
		}
		m.configsList.Items = items
		m.setRenderItem()
		m.configsList.ApplyFilter()

		m.state = stateReady
		l().Info("ConfigsView: Config list updated")
		return nil

	case TickMsg:
		l().Infof("ConfigsView: Received TickMsg, state=%v, visible=%v", m.state, m.visible)
		// Only check for changes if view is visible, ready, and not showing dialogs
		if m.visible && m.state == stateReady && !m.confirmDialog.Visible && !m.loadingView.Visible() {
			return tea.Batch(
				CheckConfigsCmd(m.lastSnapshot),
				tickCmd(),
			)
		}
		// Continue ticking even if not visible/ready
		return tickCmd()

	case configRotatedMsg:
		l().Infof("Config rotated: %s → %s", msg.Old.Config.Spec.Name, msg.New.Config.Spec.Name)
		// After rotating a config, reload the config list so the "Used" state
		// is recalculated (services may have been updated to reference the new config).
		return tea.Batch(
			loadConfigsCmd(),
			tea.Printf("Rotated %s → %s", msg.Old.Config.Spec.Name, msg.New.Config.Spec.Name),
		)

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

	case configCreatedMsg:
		l().Infof("Config created successfully: %s", msg.Config.Spec.Name)
		return loadConfigsCmd()

	case editorContentReadyMsg:
		if msg.Err != nil {
			l().Errorf("Error creating config from editor: %v", msg.Err)
			// Preserve the content and return to name entry
			m.createDialogActive = true
			m.createDialogStep = "details-inline"
			m.createConfigData = string(msg.Data)
			m.createDialogError = msg.Err.Error()
			m.createInputFocus = 0
			m.createNameInput.Focus()
			return nil
		}
		// Success
		return loadConfigsCmd()

	case fileContentReadyMsg:
		if msg.Err != nil {
			l().Errorf("Error creating config from file: %v", msg.Err)
			// Preserve the file path and return to name entry
			m.createDialogActive = true
			m.createDialogStep = "details-file"
			m.createFileInput.SetValue(msg.FilePath)
			m.createDialogError = msg.Err.Error()
			m.createInputFocus = 0
			m.createNameInput.Focus()
			return nil
		}
		// Success
		return loadConfigsCmd()

	case configCreateErrorMsg:
		l().Errorf("Error creating config: %v", msg.err)
		// Return to create dialog with error message
		m.createDialogActive = true
		m.createDialogError = msg.err.Error()
		m.fileBrowserActive = false
		return nil

	case filesLoadedMsg:
		if msg.Error != nil {
			l().Errorf("Error loading files: %v", msg.Error)
			m.fileBrowserActive = false
			m.createDialogActive = true
			m.createDialogError = fmt.Sprintf("Failed to load directory: %v", msg.Error)
			return nil
		}
		m.fileBrowserPath = msg.Path
		m.fileBrowserFiles = msg.Files
		m.fileBrowserCursor = 0
		m.fileBrowserActive = true // Ensure browser stays active
		return nil

	case usedByMsg:
		if msg.Error != nil {
			l().Errorf("Error getting used by stacks: %v", msg.Error)
			m.errorDialogActive = true
			m.err = msg.Error
			return nil
		}
		l().Infof("Config %s is used by %d service(s)", msg.ConfigName, len(msg.UsedBy))

		// Initialize usedByList with a new viewport. Use sensible fallbacks
		// (model width/height) if configsList viewport hasn't been sized yet.
		w := m.configsList.Viewport.Width
		if w <= 0 {
			w = m.width
		}
		h := m.configsList.Viewport.Height
		if h <= 0 {
			if m.height > 0 {
				h = m.height - 2
			}
			if h <= 0 {
				h = 20
			}
		}
		vp := viewport.New(w, h)
		vp.SetContent("")

		m.usedByList = filterlist.FilterableList[usedByItem]{
			Viewport: vp,
			Match: func(item usedByItem, query string) bool {
				return strings.Contains(strings.ToLower(item.StackName), strings.ToLower(query)) ||
					strings.Contains(strings.ToLower(item.ServiceName), strings.ToLower(query))
			},
			RenderItem: func(item usedByItem, selected bool, _ int) string {
				line := fmt.Sprintf("%-24s %-24s", item.StackName, item.ServiceName)
				if selected {
					return ui.CursorStyle.Render(line)
				}
				itemStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("117"))
				return itemStyle.Render(line)
			},
		}

		m.usedByList.Items = msg.UsedBy
		// Keep viewport sizes in sync
		m.usedByList.Viewport.Width = vp.Width
		m.usedByList.Viewport.Height = vp.Height
		m.usedByList.ApplyFilter()

		m.usedByConfigName = msg.ConfigName
		m.usedByViewActive = true
		return nil

	case createConfigMsg:
		l().Infof("Opening editor to create config: %s", msg.Name)
		return createConfigInEditorCmd(msg.Name, []byte(m.createConfigData))

	case errorMsg:
		l().Errorf("Error occurred: %v", msg)
		m.state = stateError
		m.err = msg
		m.errorDialogActive = true
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
		if m.errorDialogActive {
			if msg.String() == "enter" || msg.String() == "esc" {
				m.errorDialogActive = false
				m.err = nil
				m.state = stateReady
				return nil
			}
			return nil
		}

		if m.createDialogActive {
			return m.handleCreateDialogKey(msg)
		}

		if m.usedByViewActive {
			return m.handleUsedByViewKey(msg)
		}

		if m.fileBrowserActive {
			return m.handleFileBrowserKey(msg)
		}

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
		// Handle specific keys in switch, then navigation keys
		switch msg.String() {
		case "ctrl+d":
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
			cfgName := m.selectedConfig()
			l().Infof("Edit key pressed for config: %s", cfgName)
			// Start editor; the editCmd will send back editConfigDoneMsg or editConfigErrorMsg
			return editConfigInEditorCmd(cfgName)
		case "u":
			cfgName := m.selectedConfig()
			if cfgName == "" {
				l().Warn("UsedBy key pressed but no config selected")
				return nil
			}
			l().Infof("UsedBy key pressed for config: %s", cfgName)
			return getUsedByStacksCmd(cfgName)
		case "n":
			l().Info("Create key pressed")
			m.createDialogActive = true
			m.createDialogStep = "source"
			m.createConfigSource = "file" // default
			m.createNameInput.SetValue("")
			m.createConfigData = ""
			m.createDialogError = ""
			return nil

		case "c":
			// Clone selected config: ask for new name, prefill editor with existing content
			cfgName := m.selectedConfig()
			if cfgName == "" {
				l().Warn("Clone key pressed but no config selected")
				return nil
			}
			l().Infof("Clone key pressed for config: %s", cfgName)
			// Inspect to get content
			ctx := context.Background()
			cfg, err := docker.InspectConfig(ctx, cfgName)
			if err != nil {
				l().Errorf("Failed to inspect config for clone: %v", err)
				m.err = err
				m.errorDialogActive = true
				return nil
			}
			// Prefill create dialog with existing content and suggested name
			suggested := cfg.Config.Spec.Name + "_clone"
			m.createDialogActive = true
			m.createDialogStep = "details-inline"
			m.createNameInput.SetValue(suggested)
			m.createConfigData = string(cfg.Data)
			m.createDialogError = ""
			m.createInputFocus = 0
			m.createNameInput.Focus()
			return nil
		case "i":
			cfg := m.selectedConfig()
			l().Infof("Inspect key pressed for config: %s", cfg)
			return inspectConfigCmd(m.selectedConfig())
		case "enter":
			cfg := m.selectedConfig()
			l().Infof("Inspect key pressed for config: %s", cfg)
			return inspectRawConfigCmd(m.selectedConfig())
		default:
			// Let FilterableList handle navigation keys (up/down/pgup/pgdown)
			m.configsList.HandleKey(msg)
			return nil
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
		createdStr := "N/A"
		if !cfg.CreatedAt.IsZero() {
			createdStr = cfg.CreatedAt.Format("2006-01-02 15:04:05")
		}
		updatedStr := "N/A"
		if !cfg.UpdatedAt.IsZero() {
			updatedStr = cfg.UpdatedAt.Format("2006-01-02 15:04:05")
		}
		// Include "CONFIG USED" column
		usedCol := len("CONFIG USED")
		usedStr := " "
		if cfg.Used {
			usedStr = "●"
		}
		line := fmt.Sprintf("%-*s        %-*s        %-*s        %-19s        %-19s", nameCol, cfg.Name, idCol, cfg.ID, usedCol, usedStr, createdStr, updatedStr)
		if selected {
			return ui.CursorStyle.Render(line)
		}
		return itemStyle.Render(line)
	}
}

func (m *Model) handleCreateDialogKey(msg tea.KeyMsg) tea.Cmd {
	switch m.createDialogStep {
	case "source":
		switch msg.String() {
		case "esc":
			m.createDialogActive = false
			m.createDialogError = ""
			return nil
		case "up", "down":
			// Toggle between file and inline
			if m.createConfigSource == "file" {
				m.createConfigSource = "inline"
			} else {
				m.createConfigSource = "file"
			}
			return nil
		case "enter":
			// Move to name entry
			m.createDialogError = "" // Clear any previous error
			if m.createConfigSource == "file" {
				m.createDialogStep = "details-file"
			} else {
				m.createDialogStep = "details-inline"
			}
			m.createInputFocus = 0
			m.createNameInput.SetValue("")
			m.createFileInput.SetValue("")
			m.createConfigData = ""
			m.createNameInput.Focus()
			m.createFileInput.Blur()
			return nil
		}

	case "details-file":
		switch msg.String() {
		case "esc":
			m.createDialogActive = false
			m.createDialogError = ""
			m.createNameInput.Blur()
			m.createFileInput.Blur()
			m.createConfigPath = ""
			m.createInputFocus = 0
			return nil
		case "tab", "shift+tab":
			// Toggle focus between name and file inputs
			if m.createInputFocus == 0 {
				m.createInputFocus = 1
				m.createNameInput.Blur()
				m.createFileInput.Focus()
			} else {
				m.createInputFocus = 0
				m.createFileInput.Blur()
				m.createNameInput.Focus()
			}
			return nil
		case "f", "F":
			// Only open file browser when focused on file input
			if m.createInputFocus == 1 {
				m.createDialogActive = false
				m.fileBrowserActive = true
				homeDir, _ := os.UserHomeDir()
				if homeDir == "" {
					homeDir = "/"
				}
				return loadFilesCmd(homeDir)
			}
			// Otherwise let textinput handle it (typing 'f')
			var cmd tea.Cmd
			if m.createInputFocus == 0 {
				m.createNameInput, cmd = m.createNameInput.Update(msg)
			} else {
				m.createFileInput, cmd = m.createFileInput.Update(msg)
			}
			if m.createDialogError != "" {
				m.createDialogError = ""
			}
			return cmd
		case "enter":
			// If there's an error, clear it and stay in editing mode
			if m.createDialogError != "" {
				m.createDialogError = ""
				return nil
			}
			// Validate name
			if m.createNameInput.Value() == "" {
				m.createDialogError = "Config name cannot be empty"
				return nil
			}
			if err := validateConfigName(m.createNameInput.Value()); err != nil {
				m.createDialogError = err.Error()
				return nil
			}
			// Validate file path
			filePath := m.createFileInput.Value()
			if filePath == "" {
				m.createDialogError = "Please enter or select a file path"
				return nil
			}
			// All valid, create config
			m.createDialogActive = false
			m.createDialogError = ""
			m.createNameInput.Blur()
			m.createFileInput.Blur()
			return createConfigFromFileCmd(m.createNameInput.Value(), filePath)
		default:
			// Pass keys to the focused textinput
			var cmd tea.Cmd
			if m.createInputFocus == 0 {
				m.createNameInput, cmd = m.createNameInput.Update(msg)
			} else {
				m.createFileInput, cmd = m.createFileInput.Update(msg)
			}
			// Clear error when user types
			if m.createDialogError != "" {
				m.createDialogError = ""
			}
			return cmd
		}
	case "details-inline":
		switch msg.String() {
		case "esc":
			m.createDialogActive = false
			m.createDialogError = ""
			m.createNameInput.Blur()
			m.createConfigData = ""
			m.createInputFocus = 0
			return nil
		case "tab", "shift+tab":
			// Toggle focus between name and content
			if m.createInputFocus == 0 {
				m.createInputFocus = 1
				m.createNameInput.Blur()
			} else {
				m.createInputFocus = 0
				m.createNameInput.Focus()
			}
			return nil
		case "e", "E":
			// Only open editor when focused on content field
			if m.createInputFocus == 1 {
				if m.createNameInput.Value() == "" {
					m.createDialogError = "Please enter a config name first"
					return nil
				}
				if err := validateConfigName(m.createNameInput.Value()); err != nil {
					m.createDialogError = err.Error()
					return nil
				}
				m.createDialogActive = false
				m.createNameInput.Blur()
				return createConfigInEditorCmd(m.createNameInput.Value(), []byte(m.createConfigData))
			}
			// Otherwise let textinput handle it (typing 'e')
			var cmd tea.Cmd
			m.createNameInput, cmd = m.createNameInput.Update(msg)
			if m.createDialogError != "" {
				m.createDialogError = ""
			}
			return cmd
		case "enter":
			// If there's an error, clear it and stay in editing mode
			if m.createDialogError != "" {
				m.createDialogError = ""
				return nil
			}
			// Validate name
			if m.createNameInput.Value() == "" {
				m.createDialogError = "Config name cannot be empty"
				return nil
			}
			if err := validateConfigName(m.createNameInput.Value()); err != nil {
				m.createDialogError = err.Error()
				return nil
			}
			// Check if we have data
			if m.createConfigData == "" {
				m.createDialogError = "Please add content in editor (press Tab then E)"
				return nil
			}
			// All valid, create config with existing data
			m.createDialogActive = false
			m.createDialogError = ""
			m.createNameInput.Blur()
			return createConfigInEditorCmd(m.createNameInput.Value(), []byte(m.createConfigData))
		default:
			// Pass keys to name input only when focused on it
			if m.createInputFocus == 0 {
				var cmd tea.Cmd
				m.createNameInput, cmd = m.createNameInput.Update(msg)
				// Clear error when user types
				if m.createDialogError != "" {
					m.createDialogError = ""
				}
				return cmd
			}
			return nil
		}
	}

	return nil
}

func (m *Model) handleFileBrowserKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.fileBrowserActive = false
		m.createDialogActive = true
		return nil

	case "up":
		if m.fileBrowserCursor > 0 {
			m.fileBrowserCursor--
		}
		return nil

	case "down":
		if m.fileBrowserCursor < len(m.fileBrowserFiles)-1 {
			m.fileBrowserCursor++
		}
		return nil

	case "pgup":
		m.fileBrowserCursor -= 10
		if m.fileBrowserCursor < 0 {
			m.fileBrowserCursor = 0
		}
		return nil

	case "pgdown":
		m.fileBrowserCursor += 10
		if m.fileBrowserCursor >= len(m.fileBrowserFiles) {
			m.fileBrowserCursor = len(m.fileBrowserFiles) - 1
		}
		return nil

	case "enter":
		if len(m.fileBrowserFiles) == 0 {
			return nil
		}

		selected := m.fileBrowserFiles[m.fileBrowserCursor]

		// Handle parent directory
		if selected == ".." {
			parentDir := filepath.Dir(m.fileBrowserPath)
			if parentDir == m.fileBrowserPath {
				parentDir = "/"
			}
			return loadFilesCmd(parentDir)
		}

		// Handle directory
		if strings.HasSuffix(selected, "/") {
			dirPath := strings.TrimSuffix(selected, "/")
			return loadFilesCmd(dirPath)
		}

		// It's a file - set the path and close the file browser
		m.createConfigPath = selected
		m.createFileInput.SetValue(selected)
		m.fileBrowserActive = false

		// If a name is already provided and valid, create immediately
		name := m.createNameInput.Value()
		if name != "" {
			if err := validateConfigName(name); err == nil {
				m.createDialogActive = false
				m.createDialogError = ""
				m.createNameInput.Blur()
				m.createFileInput.Blur()
				return createConfigFromFileCmd(name, selected)
			}
		}

		// Otherwise return focus to create dialog so user can enter a name
		m.createDialogActive = true
		m.createDialogStep = "details-file"
		m.createInputFocus = 1
		m.createFileInput.Focus()
		return nil
	}
	return nil
}

func (m *Model) handleUsedByViewKey(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		// Go back to configs view
		m.usedByViewActive = false
		m.usedByList.Items = nil
		m.usedByConfigName = ""
		return nil

	case "enter":
		// Navigate to the services in the selected stack
		if len(m.usedByList.Filtered) == 0 {
			return nil
		}
		selectedStack := m.usedByList.Filtered[m.usedByList.Cursor].StackName
		l().Infof("Navigating to services in stack: %s", selectedStack)
		m.usedByViewActive = false
		m.usedByList.Items = nil
		m.usedByConfigName = ""
		return func() tea.Msg {
			// Send a generic navigation message with a payload for services view.
			// Use Replace=false to indicate this should be pushed onto the view stack.
			payload := map[string]interface{}{"stackName": selectedStack}
			return view.NavigateToMsg{ViewName: "services", Payload: payload, Replace: false}
		}

	default:
		// Handle navigation in the used by list
		if m.usedByList.Mode == filterlist.ModeSearching {
			m.usedByList.HandleKey(msg)
		} else {
			m.usedByList.HandleKey(msg)
		}
		return nil
	}
}
