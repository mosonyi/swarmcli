package contexts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/confirmdialog"
	helpview "swarmcli/views/help"
	"swarmcli/views/view"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type RefreshTickMsg time.Time

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return RefreshTickMsg(t)
	})
}

// StartTickerCmd starts the periodic refresh ticker
func StartTickerCmd() tea.Cmd {
	return tickCmd()
}

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case RefreshTickMsg:
		// Auto-refresh contexts list every 5 seconds when visible and no dialogs open
		if m.Visible && !m.HasActiveDialog() && !m.loading {
			return tea.Batch(
				func() tea.Msg { return LoadContextsCmd() },
				tickCmd(),
			)
		}
		// Continue ticking even if not visible
		return tickCmd()

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		// Keep the internal list viewport in sync with the new size so
		// the framed box fills the area immediately.
		if msg.Width > 0 {
			m.List.Viewport.Width = msg.Width
		}
		if msg.Height > 0 {
			h := msg.Height - 2
			if h <= 0 {
				h = 20
			}
			m.List.Viewport.Height = h
		}
		if !m.ready {
			m.ready = true
		}
		return nil

	case ContextsLoadedMsg:
		// Instrumentation: use package logger helper and emit compact debug JSON
		// so we can confirm delivery in user environments.
		lg := l()
		if msg.Error != nil {
			lg.Warnw("ContextsLoadedMsg received with error", "error", msg.Error)
		} else {
			lg.Infow("ContextsLoadedMsg received", "count", len(msg.Contexts))
		}

		debug := map[string]any{
			"count": len(msg.Contexts),
			"error": nil,
		}
		if msg.Error != nil {
			debug["error"] = msg.Error.Error()
		}
		if b, jerr := json.Marshal(debug); jerr == nil {
			lg.Debugf("[ContextsLoaded] %s", string(b))
		}

		if msg.Error != nil {
			m.SetError(msg.Error.Error())
			m.SetLoading(false)
			return nil
		}
		m.SetContexts(msg.Contexts)
		m.SetLoading(false)
		m.SetError("")
		return nil

	case ContextSwitchedMsg:
		m.SetSwitchPending(false)
		if !msg.Success {
			m.SetError("Failed to switch context: " + msg.Error.Error())
			m.SetSuccess("")
			m.errorDialogActive = true
			return nil
		}
		m.SetError("")
		m.SetSuccess("")
		// Refresh contexts list and then navigate to stacks view
		m.SetLoading(true)
		return tea.Batch(
			func() tea.Msg { return LoadContextsCmd() },
			func() tea.Msg { return ContextChangedNotification{} },
		)

	case ContextExportedMsg:
		if !msg.Success {
			// Check if error is file_exists
			if msg.Error != nil && msg.Error.Error() == "file_exists" {
				// Show confirmation dialog
				m.pendingExportContext = msg.ContextName
				m.pendingAction = "export"
				m.confirmDialog = m.confirmDialog.Show(
					fmt.Sprintf("File %s already exists. Overwrite?", msg.FilePath),
				)
				return nil
			}
			m.SetError("Failed to export context: " + msg.Error.Error())
			m.SetSuccess("")
			return nil
		}
		m.SetError("")
		m.SetSuccess("Exported " + msg.ContextName + " to " + msg.FilePath)
		return nil

	case ContextImportedMsg:
		m.fileBrowserActive = false
		if !msg.Success {
			m.SetError("Failed to import context: " + msg.Error.Error())
			m.SetSuccess("")
			return nil
		}
		m.SetError("")
		m.SetSuccess("Imported context: " + msg.ContextName)
		// Auto-refresh the list to show the new context
		m.SetLoading(true)
		return func() tea.Msg {
			return LoadContextsCmd()
		}

	case ContextDeletedMsg:
		if !msg.Success {
			m.SetError("Failed to delete context: " + msg.Error.Error())
			m.SetSuccess("")
			return nil
		}
		m.SetError("")
		m.SetSuccess("Deleted context: " + msg.ContextName)
		// Refresh the list
		m.SetLoading(true)
		return func() tea.Msg {
			return LoadContextsCmd()
		}

	case FilesLoadedMsg:
		if msg.Error != nil {
			m.SetError("Failed to read directory: " + msg.Error.Error())
			// Keep cert file browser open, but close import browser
			if !m.certFileBrowserActive {
				m.fileBrowserActive = false
				m.importInputActive = false
				m.importInput.Blur()
			}
			return nil
		}
		if len(msg.Files) == 0 {
			// For cert browser, empty is ok (just ".." entry)
			// For import browser, need .tar files
			if !m.certFileBrowserActive {
				m.SetError("No .tar files found in " + msg.Path)
				m.fileBrowserActive = false
				m.importInputActive = false
				m.importInput.Blur()
				return nil
			}
		}
		m.fileBrowserPath = msg.Path
		m.fileBrowserFiles = msg.Files
		m.fileBrowserCursor = 0
		// Only set fileBrowserActive if we're browsing for import (not cert files)
		if !m.certFileBrowserActive {
			m.fileBrowserActive = true
		}
		m.importInputActive = false
		m.importInput.Blur()
		return nil

	case ContextCreatedMsg:
		if !msg.Success {
			// Show error in the create dialog (which should still be open)
			m.SetError("Failed to create context: " + msg.Error.Error())
			m.SetSuccess("")
			return nil
		}
		// Success - close create dialog and clear fields
		m.createDialogActive = false
		m.createNameInput.Blur()
		m.createDescInput.Blur()
		m.createHostInput.Blur()
		m.createCAInput.Blur()
		m.createCertInput.Blur()
		m.createKeyInput.Blur()
		m.createNameInput.SetValue("")
		m.createDescInput.SetValue("")
		m.createHostInput.SetValue("")
		m.createCAInput.SetValue("")
		m.createCertInput.SetValue("")
		m.createKeyInput.SetValue("")
		m.createKeyInput.SetValue("")
		m.createTLSEnabled = false
		m.SetError("")
		m.SetSuccess("Created context: " + msg.ContextName)
		// Refresh the list to show the new context
		m.SetLoading(true)
		return func() tea.Msg {
			return LoadContextsCmd()
		}

	case ContextUpdatedMsg:
		if !msg.Success {
			// Show error in the edit dialog (which should still be open)
			m.SetError("Failed to update context: " + msg.Error.Error())
			m.SetSuccess("")
			return nil
		}
		// Success - close edit dialog and clear fields
		m.editDialogActive = false
		m.editContextName = ""
		m.editDescInput.Blur()
		m.editDescInput.SetValue("")
		m.SetError("")
		m.SetSuccess("Updated context: " + msg.ContextName)
		// Refresh the list to show the updated context
		m.SetLoading(true)
		return func() tea.Msg {
			return LoadContextsCmd()
		}

	case confirmdialog.ResultMsg:
		if msg.Confirmed {
			switch m.pendingAction {
			case "export":
				if m.pendingExportContext != "" {
					// User confirmed overwrite, export with force
					contextName := m.pendingExportContext
					m.pendingExportContext = ""
					m.pendingAction = ""
					m.confirmDialog.Hide()
					return ExportContextWithForceCmd(contextName)
				}
			case "delete":
				if m.pendingDeleteContext != "" {
					// User confirmed delete
					contextName := m.pendingDeleteContext
					m.pendingDeleteContext = ""
					m.pendingAction = ""
					m.confirmDialog.Hide()
					return DeleteContextCmd(contextName)
				}
			}
		}
		// User cancelled or no pending action
		m.pendingExportContext = ""
		m.pendingDeleteContext = ""
		m.pendingAction = ""
		m.confirmDialog.Hide()
		return nil

	case tea.KeyMsg:
		// Clear active filter with ESC (consistent with stacks view)
		if m.List.Mode != filterlist.ModeSearching && msg.Type == tea.KeyEsc && m.List.Query != "" {
			m.List.Query = ""
			m.List.Mode = filterlist.ModeNormal
			m.List.ApplyFilter()
			m.List.Cursor = 0
			m.List.Viewport.GotoTop()
			return nil
		}

		// Handle cert file browser if active (highest priority)
		if m.certFileBrowserActive {
			switch msg.String() {
			case "up", "k":
				if m.fileBrowserCursor > 0 {
					m.fileBrowserCursor--
				}
				return nil
			case "down", "j":
				if m.fileBrowserCursor < len(m.fileBrowserFiles)-1 {
					m.fileBrowserCursor++
				}
				return nil
			case "pgup":
				// Jump up 10 items
				m.fileBrowserCursor -= 10
				if m.fileBrowserCursor < 0 {
					m.fileBrowserCursor = 0
				}
				return nil
			case "pgdown":
				// Jump down 10 items
				m.fileBrowserCursor += 10
				if m.fileBrowserCursor >= len(m.fileBrowserFiles) {
					m.fileBrowserCursor = len(m.fileBrowserFiles) - 1
				}
				return nil
			case "enter":
				if len(m.fileBrowserFiles) > 0 && m.fileBrowserCursor < len(m.fileBrowserFiles) {
					selectedFile := m.fileBrowserFiles[m.fileBrowserCursor]

					// Handle parent directory navigation
					if selectedFile == ".." {
						parentDir := filepath.Dir(m.fileBrowserPath)
						m.fileBrowserPath = parentDir
						m.fileBrowserCursor = 0
						return LoadCertFilesCmd(parentDir)
					}

					// Handle directory navigation (directories end with /)
					if strings.HasSuffix(selectedFile, "/") {
						// Navigate into directory
						dirPath := strings.TrimSuffix(selectedFile, "/")
						m.fileBrowserPath = dirPath
						m.fileBrowserCursor = 0
						return LoadCertFilesCmd(dirPath)
					}

					// It's a file - select it (only for create dialog now)
					switch m.certFileTarget {
					case "ca":
						m.createCAInput.SetValue(selectedFile)
					case "cert":
						m.createCertInput.SetValue(selectedFile)
					case "key":
						m.createKeyInput.SetValue(selectedFile)
					}
					// Remember the directory for next time
					m.lastCertBrowserPath = filepath.Dir(selectedFile)
					// Close file browser and return to dialog
					m.certFileBrowserActive = false
					m.fileBrowserFiles = nil
					m.fileBrowserCursor = 0
					m.certFileTarget = ""
				}
				return nil
			case "esc":
				// Cancel file browser, return to create dialog
				m.certFileBrowserActive = false
				m.fileBrowserFiles = nil
				m.fileBrowserCursor = 0
				m.certFileTarget = ""
				return nil
			}
			return nil
		}

		// Handle create dialog if active
		if m.createDialogActive {
			switch msg.String() {
			case "enter":
				// If there's an error displayed, clear it on Enter
				if m.GetError() != "" {
					m.SetError("")
					return nil
				}

				// Submit if required fields have values
				name := strings.TrimSpace(m.createNameInput.Value())
				desc := strings.TrimSpace(m.createDescInput.Value())
				host := strings.TrimSpace(m.createHostInput.Value())
				ca := strings.TrimSpace(m.createCAInput.Value())
				cert := strings.TrimSpace(m.createCertInput.Value())
				key := strings.TrimSpace(m.createKeyInput.Value())

				if name == "" || host == "" {
					m.SetError("Both name and host are required")
					return nil
				}
				if m.createTLSEnabled {
					if ca == "" || cert == "" || key == "" {
						m.SetError("All certificate files (CA, Cert, Key) are required when TLS is enabled")
						return nil
					}
				}

				// Submit the create command but keep dialog open until we get response
				useTLS := m.createTLSEnabled
				m.SetError("")
				m.SetSuccess("")
				return CreateContextWithCertFilesCmd(name, desc, host, ca, cert, key, useTLS)
			case "esc":
				// Cancel create
				m.createDialogActive = false
				m.createNameInput.Blur()
				m.createDescInput.Blur()
				m.createHostInput.Blur()
				m.createCAInput.Blur()
				m.createCertInput.Blur()
				m.createKeyInput.Blur()
				m.createNameInput.SetValue("")
				m.createDescInput.SetValue("")
				m.createHostInput.SetValue("")
				m.createCAInput.SetValue("")
				m.createCertInput.SetValue("")
				m.createKeyInput.SetValue("")
				m.createTLSEnabled = false
				m.SetError("")
				m.SetSuccess("")
				return nil
			case "f":
				// Open file browser for cert file selection (only if focused on cert inputs)
				if m.createInputFocus >= 4 && m.createInputFocus <= 6 && m.createTLSEnabled {
					// Determine which cert file is being browsed
					switch m.createInputFocus {
					case 4:
						m.certFileTarget = "ca"
					case 5:
						m.certFileTarget = "cert"
					case 6:
						m.certFileTarget = "key"
					}
					// Get current input value to determine starting directory
					currentPath := ""
					switch m.certFileTarget {
					case "ca":
						currentPath = m.createCAInput.Value()
					case "cert":
						currentPath = m.createCertInput.Value()
					case "key":
						currentPath = m.createKeyInput.Value()
					}
					// If no path, use last browsed directory or default to home
					if currentPath == "" {
						if m.lastCertBrowserPath != "" {
							currentPath = m.lastCertBrowserPath
						} else {
							homeDir, err := os.UserHomeDir()
							if err != nil || homeDir == "" || homeDir == "/" {
								// Fallback: try common home directory patterns
								currentPath = "/home/vscode" // devcontainer default
								if _, err := os.Stat(currentPath); err != nil {
									currentPath = "/tmp"
								}
							} else {
								currentPath = homeDir
							}
						}
					} else {
						// Use directory of current path
						currentPath = filepath.Dir(currentPath)
					}
					m.certFileBrowserActive = true
					m.fileBrowserPath = currentPath
					return LoadCertFilesCmd(currentPath)
				}
				// If not on cert fields, pass 'f' to the textinput
				var cmd tea.Cmd
				switch m.createInputFocus {
				case 0:
					m.createNameInput, cmd = m.createNameInput.Update(msg)
				case 1:
					m.createDescInput, cmd = m.createDescInput.Update(msg)
				case 2:
					m.createHostInput, cmd = m.createHostInput.Update(msg)
				}
				return cmd
			case "tab", "shift+tab", "down":
				// Move focus forward
				m.createInputFocus++
				if m.createInputFocus > 6 {
					m.createInputFocus = 0
				}
				// Skip cert file inputs if TLS not enabled
				if m.createInputFocus >= 4 && m.createInputFocus <= 6 && !m.createTLSEnabled {
					m.createInputFocus = 0
				}
				m.updateCreateFocus()
				return nil
			case "up":
				// Move focus backward
				m.createInputFocus--
				if m.createInputFocus < 0 {
					if m.createTLSEnabled {
						m.createInputFocus = 6 // Key input
					} else {
						m.createInputFocus = 3 // TLS checkbox
					}
				}
				// Skip cert file inputs if TLS not enabled
				if m.createInputFocus >= 4 && m.createInputFocus <= 6 && !m.createTLSEnabled {
					m.createInputFocus = 3
				}
				m.updateCreateFocus()
				return nil
			case " ":
				// Toggle TLS checkbox if focused on it
				if m.createInputFocus == 3 {
					m.createTLSEnabled = !m.createTLSEnabled
					return nil
				}
				// Otherwise pass to textinput
				fallthrough
			default:
				// Update the focused textinput
				var cmd tea.Cmd
				switch m.createInputFocus {
				case 0:
					m.createNameInput, cmd = m.createNameInput.Update(msg)
				case 1:
					m.createDescInput, cmd = m.createDescInput.Update(msg)
				case 2:
					m.createHostInput, cmd = m.createHostInput.Update(msg)
				case 4:
					m.createCAInput, cmd = m.createCAInput.Update(msg)
				case 5:
					m.createCertInput, cmd = m.createCertInput.Update(msg)
				case 6:
					m.createKeyInput, cmd = m.createKeyInput.Update(msg)
				}
				return cmd
			}
		}

		// Handle edit dialog if active
		if m.editDialogActive {
			switch msg.String() {
			case "enter":
				// If there's an error displayed, clear it on Enter
				if m.GetError() != "" {
					m.SetError("")
					return nil
				}

				// Get description value
				description := strings.TrimSpace(m.editDescInput.Value())

				// Update only the description (no host or cert changes)
				return UpdateContextDescriptionCmd(m.editContextName, description)

			case "esc":
				// Cancel edit dialog
				m.editDialogActive = false
				m.editContextName = ""
				m.editDescInput.Blur()
				m.editDescInput.SetValue("")
				m.SetError("")
				return nil

			default:
				// Update the description textinput
				var cmd tea.Cmd
				m.editDescInput, cmd = m.editDescInput.Update(msg)
				return cmd
			}
		}

		// Handle error dialog if active
		if m.errorDialogActive {
			if msg.String() == "enter" || msg.String() == "esc" {
				m.errorDialogActive = false
				m.SetError("")
			}
			return nil
		}

		// Handle file browser if active
		if m.fileBrowserActive {
			switch msg.String() {
			case "up", "k":
				if m.fileBrowserCursor > 0 {
					m.fileBrowserCursor--
				}
				return nil
			case "down", "j":
				if m.fileBrowserCursor < len(m.fileBrowserFiles)-1 {
					m.fileBrowserCursor++
				}
				return nil
			case "pgup":
				// Jump up 10 items
				m.fileBrowserCursor -= 10
				if m.fileBrowserCursor < 0 {
					m.fileBrowserCursor = 0
				}
				return nil
			case "pgdown":
				// Jump down 10 items
				m.fileBrowserCursor += 10
				if m.fileBrowserCursor >= len(m.fileBrowserFiles) {
					m.fileBrowserCursor = len(m.fileBrowserFiles) - 1
				}
				return nil
			case "enter":
				// Select file and import, or navigate into directory
				if m.fileBrowserCursor < len(m.fileBrowserFiles) {
					selectedFile := m.fileBrowserFiles[m.fileBrowserCursor]

					// Handle parent directory navigation
					if selectedFile == ".." {
						parentDir := filepath.Dir(m.fileBrowserPath)
						m.fileBrowserPath = parentDir
						m.fileBrowserCursor = 0
						return LoadFilesCmd(parentDir)
					}

					// Handle directory navigation (directories end with /)
					if strings.HasSuffix(selectedFile, "/") {
						// Navigate into directory
						dirPath := strings.TrimSuffix(selectedFile, "/")
						m.fileBrowserPath = dirPath
						m.fileBrowserCursor = 0
						return LoadFilesCmd(dirPath)
					}

					// It's a .tar file - import it
					m.fileBrowserActive = false
					m.fileBrowserFiles = []string{}
					m.SetError("")
					m.SetSuccess("")
					return ImportContextCmd(selectedFile)
				}
				return nil
			case "esc":
				// Cancel file browser
				m.fileBrowserActive = false
				m.fileBrowserFiles = []string{}
				return nil
			default:
				return nil
			}
		}

		// Handle import input if active
		if m.importInputActive {
			switch msg.String() {
			case "enter":
				// Submit directory path to load tar files
				dirPath := strings.TrimSpace(m.importInput.Value())
				if dirPath == "" {
					dirPath = "/tmp"
				}
				m.SetError("")
				m.SetSuccess("")
				return LoadFilesCmd(dirPath)
			case "esc":
				// Cancel import
				m.importInputActive = false
				m.importInput.Blur()
				m.importInput.SetValue("")
				return nil
			default:
				// Update textinput
				var cmd tea.Cmd
				m.importInput, cmd = m.importInput.Update(msg)
				return cmd
			}
		}

		// Handle confirm dialog if visible
		if m.confirmDialog.Visible {
			// Intercept ESC to just close dialog without navigating back
			if msg.String() == "esc" {
				m.pendingExportContext = ""
				m.pendingDeleteContext = ""
				m.pendingAction = ""
				m.confirmDialog.Hide()
				return nil
			}
			return m.confirmDialog.Update(msg)
		}

		if m.IsSwitchPending() {
			return nil
		}

		switch msg.String() {
		case "up", "k":
			m.MoveCursor(-1)
			return nil

		case "down", "j":
			m.MoveCursor(1)
			return nil

		case "enter":
			// Switch to selected context
			ctx, ok := m.GetSelectedContext()
			if !ok {
				return nil
			}
			// Don't switch if already current
			if ctx.Current {
				return nil
			}
			m.SetSwitchPending(true)
			m.SetError("")
			m.SetSuccess("")
			return SwitchContextCmd(ctx.Name)

		case "i":
			// Inspect selected context
			ctx, ok := m.GetSelectedContext()
			if !ok {
				return nil
			}
			return InspectContextCmd(ctx.Name)

		case "?":
			return func() tea.Msg {
				return view.NavigateToMsg{
					ViewName: view.NameHelp,
					Payload:  GetContextsHelpContent(),
				}
			}

		case "N":
			if m.sortField == SortByName {
				m.sortAscending = !m.sortAscending
			} else {
				m.sortField = SortByName
				m.sortAscending = true
			}
			m.applySorting()
			return nil

		case "D":
			if m.sortField == SortByDescription {
				m.sortAscending = !m.sortAscending
			} else {
				m.sortField = SortByDescription
				m.sortAscending = true
			}
			m.applySorting()
			return nil

		case "E":
			if m.sortField == SortByEndpoint {
				m.sortAscending = !m.sortAscending
			} else {
				m.sortField = SortByEndpoint
				m.sortAscending = true
			}
			m.applySorting()
			return nil

		case "S":
			if m.sortField == SortByStatus {
				m.sortAscending = !m.sortAscending
			} else {
				m.sortField = SortByStatus
				m.sortAscending = true
			}
			m.applySorting()
			return nil

		case "x":
			// Export selected context
			ctx, ok := m.GetSelectedContext()
			if !ok {
				return nil
			}
			return ExportContextCmd(ctx.Name)

		case "m":
			// Import context from file - open file browser
			homeDir := "/tmp"
			if home, err := os.UserHomeDir(); err == nil {
				homeDir = home
			}
			m.fileBrowserActive = true
			m.SetError("")
			m.SetSuccess("")
			return LoadFilesCmd(homeDir)

		case "e":
			// Edit selected context (description only)
			ctx, ok := m.GetSelectedContext()
			if !ok {
				return nil
			}
			// Open edit dialog with current description
			m.editDialogActive = true
			m.editContextName = ctx.Name
			m.editDescInput.Focus()
			m.editDescInput.SetValue(ctx.Description)
			m.SetError("")
			m.SetSuccess("")
			return textinput.Blink

		case "c":
			// Create new context - show create dialog
			m.createDialogActive = true
			m.createInputFocus = 0
			m.createTLSEnabled = false
			m.createNameInput.Focus()
			m.createDescInput.Blur()
			m.createHostInput.Blur()
			m.createCAInput.Blur()
			m.createCertInput.Blur()
			m.createKeyInput.Blur()
			m.createNameInput.SetValue("")
			m.createDescInput.SetValue("")
			m.createHostInput.SetValue("")
			m.createCAInput.SetValue("")
			m.createCertInput.SetValue("")
			m.createKeyInput.SetValue("")
			m.SetError("")
			m.SetSuccess("")
			return textinput.Blink

		case "d":
			// Delete selected context
			ctx, ok := m.GetSelectedContext()
			if !ok {
				return nil
			}
			// Don't allow deleting current context
			if ctx.Current {
				m.SetError("Cannot delete the current context")
				m.SetSuccess("")
				return nil
			}
			// Show confirmation dialog
			m.pendingDeleteContext = ctx.Name
			m.pendingAction = "delete"
			m.confirmDialog = m.confirmDialog.Show(
				fmt.Sprintf("Delete context '%s'?", ctx.Name),
			)
			return nil
		}
	}

	return nil
}

// GetContextsHelpContent returns categorized help for the contexts view
func GetContextsHelpContent() []helpview.HelpCategory {
	return []helpview.HelpCategory{
		{
			Title: "General",
			Items: []helpview.HelpItem{
				{Keys: "<enter>", Description: "Switch to context"},
				{Keys: "<i>", Description: "Inspect context"},
				{Keys: "<c>", Description: "Create new context"},
				{Keys: "<e>", Description: "Edit context description"},
				{Keys: "<x>", Description: "Export context"},
				{Keys: "<m>", Description: "Import context from file"},
				{Keys: "<d>", Description: "Delete context"},
				{Keys: "</>", Description: "Filter"},
			},
		},
		{
			Title: "View",
			Items: []helpview.HelpItem{
				{Keys: "<shift+n>", Description: "Order by Name"},
				{Keys: "<shift+d>", Description: "Order by Description"},
				{Keys: "<shift+e>", Description: "Order by Endpoint"},
				{Keys: "<shift+s>", Description: "Order by Status"},
			},
		},
		{
			Title: "Navigation",
			Items: []helpview.HelpItem{
				{Keys: "<↑/↓>", Description: "Navigate"},
				{Keys: "<pgup>", Description: "Page up"},
				{Keys: "<pgdown>", Description: "Page down"},
				{Keys: "<esc>", Description: "Back to stacks"},
			},
		},
	}
}

// applySorting applies the current sort configuration to the filtered list
func (m *Model) applySorting() {
	if len(m.List.Filtered) == 0 {
		return
	}

	// Remember cursor position
	cursorName := ""
	if m.List.Cursor < len(m.List.Filtered) {
		cursorName = m.List.Filtered[m.List.Cursor].Name
	}

	// Sort the filtered list
	switch m.sortField {
	case SortByName:
		sort.Slice(m.List.Filtered, func(i, j int) bool {
			if m.sortAscending {
				return m.List.Filtered[i].Name < m.List.Filtered[j].Name
			}
			return m.List.Filtered[i].Name > m.List.Filtered[j].Name
		})
	case SortByDescription:
		sort.Slice(m.List.Filtered, func(i, j int) bool {
			if m.sortAscending {
				return m.List.Filtered[i].Description < m.List.Filtered[j].Description
			}
			return m.List.Filtered[i].Description > m.List.Filtered[j].Description
		})
	case SortByStatus:
		sort.Slice(m.List.Filtered, func(i, j int) bool {
			// Sort by current status (true/false)
			if m.sortAscending {
				if m.List.Filtered[i].Current == m.List.Filtered[j].Current {
					return m.List.Filtered[i].Name < m.List.Filtered[j].Name
				}
				return m.List.Filtered[i].Current
			}
			if m.List.Filtered[i].Current == m.List.Filtered[j].Current {
				return m.List.Filtered[i].Name > m.List.Filtered[j].Name
			}
			return m.List.Filtered[i].Current
		})
	case SortByEndpoint:
		sort.Slice(m.List.Filtered, func(i, j int) bool {
			if m.sortAscending {
				return m.List.Filtered[i].DockerHost < m.List.Filtered[j].DockerHost
			}
			return m.List.Filtered[i].DockerHost > m.List.Filtered[j].DockerHost
		})
	}

	// Restore cursor position
	if cursorName != "" {
		for i, c := range m.List.Filtered {
			if c.Name == cursorName {
				m.List.Cursor = i
				return
			}
		}
	}

	m.List.Cursor = 0
	m.List.Viewport.GotoTop()
}
