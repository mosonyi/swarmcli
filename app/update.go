package app

import (
	"fmt"
	"strings"
	"swarmcli/commands/api"
	"swarmcli/views/commandinput"
	loadingview "swarmcli/views/loading"
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
		cmd := m.switchToView(msg.ViewName, msg.Payload)
		return m, cmd

	case tea.WindowSizeMsg:
		cmd := m.updateForResize(msg)
		return m, cmd

	case tea.KeyMsg:
		if msg.String() == ":" {
			if !m.commandInput.Visible() {
				cmd := m.commandInput.Show()
				return m, cmd
			}
			// If already visible, consume it and do nothing
			return m, nil
		}

		// If command input is visible, forward all keys to it exclusively
		if m.commandInput.Visible() {
			var cmd tea.Cmd
			cmd = m.commandInput.Update(msg)
			return m, cmd
		}

		return m.handleKey(msg)

	case tickMsg:
		return m.handleTick(msg)

	case systeminfoview.Msg:
		var cmd tea.Cmd
		cmd = m.systemInfo.Update(msg)
		return m, cmd

	default:
		cmd := m.delegateToCurrentView(msg)
		return m, cmd
	}
}

func (m *Model) delegateToCurrentView(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	cmd = m.currentView.Update(msg)

	var vpCmd tea.Cmd
	vpCmd = m.updateViewports(msg)

	return tea.Batch(cmd, vpCmd)
}

func (m *Model) updateForResize(msg tea.WindowSizeMsg) tea.Cmd {
	var cmd tea.Cmd
	usableWidth := msg.Width - 4
	usableHeight := msg.Height - 10

	m.viewport.Width = usableWidth
	m.viewport.Height = usableHeight

	cmd = handleViewResize(m.currentView, usableWidth, usableHeight)
	return cmd
}

func handleViewResize(view view.View, width, height int) tea.Cmd {
	var adjustedMsg = tea.WindowSizeMsg{
		Width:  width,
		Height: height - systeminfoview.Height,
	}

	var cmd tea.Cmd
	cmd = view.Update(adjustedMsg)
	return cmd
}

func (m *Model) updateViewports(msg tea.Msg) tea.Cmd {
	var cmd1 tea.Cmd
	m.viewport, cmd1 = m.viewport.Update(msg)
	return cmd1
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global escape / quit handler
	if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEsc || msg.String() == "q" {
		var cmd tea.Cmd
		cmd = m.goBack()
		return m, cmd
	}

	var cmd tea.Cmd
	cmd = m.currentView.Update(msg)
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
	var resizeCmd tea.Cmd
	resizeCmd = handleViewResize(m.currentView, m.viewport.Width, m.viewport.Height)

	// Execute all lifecycle commands
	return tea.Batch(exitCmd, enterCmd, resizeCmd)
}
