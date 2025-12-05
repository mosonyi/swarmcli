package contexts

import (
	"fmt"
	"strings"
	"swarmcli/views/confirmdialog"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width
		m.viewport.Height = msg.Height
		if !m.ready {
			m.ready = true
		}
		return nil

	case ContextsLoadedMsg:
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
			return nil
		}
		m.SetError("")
		m.SetSuccess("")
		// Navigate to stacks view after successful context switch
		return func() tea.Msg {
			return ContextChangedNotification{}
		}

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
		if !msg.Success {
			m.SetError("Failed to import context: " + msg.Error.Error())
			m.SetSuccess("")
			return nil
		}
		m.SetError("")
		m.SetSuccess("Imported context: " + msg.ContextName)
		// Refresh the list to show the new context
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
			m.fileBrowserActive = false
			m.importInputActive = false
			m.importInput.Blur()
			return nil
		}
		if len(msg.Files) == 0 {
			m.SetError("No .tar files found in " + msg.Path)
			m.fileBrowserActive = false
			m.importInputActive = false
			m.importInput.Blur()
			return nil
		}
		m.fileBrowserPath = msg.Path
		m.fileBrowserFiles = msg.Files
		m.fileBrowserCursor = 0
		m.fileBrowserActive = true
		m.importInputActive = false
		m.importInput.Blur()
		return nil

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
			case "enter":
				// Select file and import
				if m.fileBrowserCursor < len(m.fileBrowserFiles) {
					selectedFile := m.fileBrowserFiles[m.fileBrowserCursor]
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

		case "e":
			// Export selected context
			ctx, ok := m.GetSelectedContext()
			if !ok {
				return nil
			}
			return ExportContextCmd(ctx.Name)

		case "f":
			// Import context from file - show input dialog for directory
			m.importInputActive = true
			m.importInput.Focus()
			m.importInput.SetValue("/tmp")
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

		case "r":
			// Refresh contexts list
			m.SetLoading(true)
			m.SetError("")
			m.SetSuccess("")
			return func() tea.Msg {
				return LoadContextsCmd()
			}
		}
	}

	return nil
}
