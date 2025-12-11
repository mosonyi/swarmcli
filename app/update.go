package app

import (
	"fmt"
	"strings"
	"swarmcli/commands/api"
	"swarmcli/views/commandinput"
	contextsview "swarmcli/views/contexts"
	loadingview "swarmcli/views/loading"
	logsview "swarmcli/views/logs"
	stacksview "swarmcli/views/stacks"
	systeminfoview "swarmcli/views/systeminfo"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case snapshotLoadedMsg:
		if msg.Err != nil {
			// Replace with error message in the loading view
			cmd := m.replaceView(loadingview.ViewName, fmt.Sprintf("Error loading snapshot: %v", msg.Err))
			return m, cmd
		}
		// Replace loading with stacks view
		cmd := m.replaceView(stacksview.ViewName, nil)
		return m, cmd
	case commandinput.SubmitMsg:
		raw := strings.TrimSpace(msg.Command)
		if raw == "" {
			return m, nil
		}

		cmd, parsedArgs, err := api.ParseInput(raw)
		if err != nil {
			m.commandInput.ShowError(err.Error())
			return m, nil
		}

		ctx := api.Context{App: &m}

		return m, cmd.Execute(ctx, parsedArgs)

	case view.NavigateToMsg:
		// Use Replace flag to decide whether to replace current view
		if msg.Replace {
			cmd := m.replaceView(msg.ViewName, msg.Payload)
			return m, cmd
		}
		cmd := m.switchToView(msg.ViewName, msg.Payload)
		return m, cmd

	case tea.WindowSizeMsg:
		cmd := m.updateForResize(msg)
		return m, cmd

	case logsview.FullscreenToggledMsg:
		// Trigger a resize to recalculate the available space
		cmd := m.updateForResize(tea.WindowSizeMsg{
			Width:  m.terminalWidth,
			Height: m.terminalHeight,
		})
		return m, cmd

	case tea.KeyMsg:
		if msg.String() == ":" {
			// Check if current view has an active dialog - if so, don't intercept
			if viewWithDialog, ok := m.currentView.(interface {
				HasActiveDialog() bool
			}); ok {
				if viewWithDialog.HasActiveDialog() {
					// Let the view handle it
					cmd := m.currentView.Update(msg)
					return m, cmd
				}
			}

			if !m.commandInput.Visible() {
				cmd := m.commandInput.Show()
				return m, cmd
			}
			// If already visible, consume it and do nothing
			return m, nil
		}

		// If command input is visible, forward all keys to it exclusively
		if m.commandInput.Visible() {
			cmd := m.commandInput.Update(msg)
			return m, cmd
		}

		return m.handleKey(msg)

	case tickMsg:
		return m.handleTick(msg)

	case systeminfoview.Msg:
		var cmd tea.Cmd
		cmd = m.systemInfo.Update(msg)
		return m, cmd

	case systeminfoview.SlowStatusMsg:
		var cmd tea.Cmd
		cmd = m.systemInfo.Update(msg)
		return m, cmd

	case systeminfoview.TickMsg:
		var cmd tea.Cmd
		cmd = m.systemInfo.Update(msg)
		return m, cmd

	case systeminfoview.SpinnerTickMsg:
		var cmd tea.Cmd
		cmd = m.systemInfo.Update(msg)
		return m, cmd

	case contextsview.ContextChangedNotification:
		// Context has changed, replace contexts view with stacks view (don't add to history)
		// and refresh system info
		cmd := m.replaceView(stacksview.ViewName, nil)
		return m, tea.Batch(
			systeminfoview.LoadStatus(),
			cmd,
		)

	case loadingview.ErrorDismissedMsg:
		// Navigate to contexts view from loading error screen
		cmd := m.replaceView(contextsview.ViewName, nil)
		return m, tea.Batch(
			cmd,
			func() tea.Msg {
				return contextsview.LoadContextsCmd()
			},
		)

	// Navigation from views should now use the generic view.NavigateToMsg
	// with the Replace flag set appropriately; specialized navigation
	// messages have been removed.

	default:
		cmd := m.delegateToCurrentView(msg)
		return m, cmd
	}
}

func (m *Model) delegateToCurrentView(msg tea.Msg) tea.Cmd {
	cmd := m.currentView.Update(msg)

	vpCmd := m.updateViewports(msg)

	return tea.Batch(cmd, vpCmd)
}

func (m *Model) updateForResize(msg tea.WindowSizeMsg) tea.Cmd {
	var cmd tea.Cmd

	// Store terminal dimensions
	m.terminalWidth = msg.Width
	m.terminalHeight = msg.Height

	// Check if current view is in fullscreen mode
	isFullscreen := false
	if logsView, ok := m.currentView.(interface{ GetFullscreen() bool }); ok {
		isFullscreen = logsView.GetFullscreen()
	}

	var usableWidth, usableHeight int
	if isFullscreen {
		// In fullscreen, use almost all space (just leave room for borders)
		usableWidth = msg.Width
		usableHeight = msg.Height - 2 // Just for top/bottom borders
	} else {
		// Normal mode: leave room for helpbar, systeminfo, etc.
		usableWidth = msg.Width - 4
		usableHeight = msg.Height - 10
	}

	m.viewport.Width = usableWidth
	m.viewport.Height = usableHeight

	cmd = handleViewResize(m.currentView, usableWidth, usableHeight, isFullscreen)
	return cmd
}

func handleViewResize(view view.View, width, height int, isFullscreen bool) tea.Cmd {
	var adjustedHeight int
	if isFullscreen {
		// In fullscreen, subtract 1 for title line
		adjustedHeight = height - 1
	} else {
		// Normal mode: subtract systeminfo height
		adjustedHeight = height - systeminfoview.Height
	}

	var adjustedMsg = tea.WindowSizeMsg{
		Width:  width,
		Height: adjustedHeight,
	}

	cmd := view.Update(adjustedMsg)
	return cmd
}

func (m *Model) updateViewports(msg tea.Msg) tea.Cmd {
	var cmd1 tea.Cmd
	m.viewport, cmd1 = m.viewport.Update(msg)
	return cmd1
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If current view has an active dialog, forward keys to it first
	if viewWithDialog, ok := m.currentView.(interface{ HasActiveDialog() bool }); ok {
		if viewWithDialog.HasActiveDialog() {
			cmd := m.currentView.Update(msg)
			return m, cmd
		}
	}
	// Check if current view is in fullscreen or search mode before handling global esc
	if msg.Type == tea.KeyEsc || msg.String() == "esc" {
		// Check if logs view has dialog open
		if logsView, ok := m.currentView.(interface {
			GetNodeSelectVisible() bool
		}); ok {
			if logsView.GetNodeSelectVisible() {
				// Let the view handle esc to close the dialog
				cmd := m.currentView.Update(msg)
				return m, cmd
			}
		}
		// Check if logs view is in fullscreen or search mode
		if logsView, ok := m.currentView.(interface {
			GetFullscreen() bool
			GetSearchMode() bool
		}); ok {
			if logsView.GetFullscreen() || logsView.GetSearchMode() {
				// Let the view handle esc to exit fullscreen or search mode
				cmd := m.currentView.Update(msg)
				return m, cmd
			}
		}
		// Check if contexts view has an active dialog
		if contextsView, ok := m.currentView.(interface {
			HasActiveDialog() bool
		}); ok {
			if contextsView.HasActiveDialog() {
				// Let the view handle esc to close the dialog
				cmd := m.currentView.Update(msg)
				return m, cmd
			}
		}
		// Check if configs view is in UsedBy view
		if configsView, ok := m.currentView.(interface {
			IsInUsedByView() bool
		}); ok {
			if configsView.IsInUsedByView() {
				// Let the configs view handle esc to close UsedBy view
				cmd := m.currentView.Update(msg)
				return m, cmd
			}
		}
		// Check if services view is in stack services mode
		if servicesView, ok := m.currentView.(interface {
			IsInStackServicesView() bool
		}); ok {
			if servicesView.IsInStackServicesView() {
				// Let the services view handle esc to go back to stacks
				cmd := m.currentView.Update(msg)
				return m, cmd
			}
		}
		// Otherwise, go back as normal
		cmd := m.goBack()
		return m, cmd
	}

	// Global quit handler
	if msg.Type == tea.KeyCtrlC || msg.String() == "q" {
		cmd := m.goBack()
		return m, cmd
	}

	cmd := m.currentView.Update(msg)
	return m, cmd
}

func (m *Model) goBack() tea.Cmd {
	// If no parent view exists → quit the app
	if m.viewStack.Len() == 0 {
		exitCmd := m.currentView.OnExit()
		return tea.Batch(exitCmd, tea.Quit)
	}

	// The view being left
	oldView := m.currentView
	exitCmd := oldView.OnExit()

	// Pop the previous view
	m.currentView = m.viewStack.Pop()

	// The view you are returning to
	enterCmd := m.currentView.OnEnter()

	// Optionally notify the view about terminal size again
	resizeCmd := handleViewResize(m.currentView, m.viewport.Width, m.viewport.Height, false)

	// Execute all lifecycle commands
	return tea.Batch(exitCmd, enterCmd, resizeCmd)
}
