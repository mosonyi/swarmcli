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
			return m.replaceView(loadingview.ViewName, fmt.Sprintf("Error loading snapshot: %v", msg.Err))
		}
		// Replace loading with stacks view
		return m.replaceView(stacksview.ViewName, nil)
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
		return m.switchToView(msg.ViewName, msg.Payload)

	case tea.WindowSizeMsg:
		return m.updateForResize(msg)

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
			m.commandInput, cmd = m.commandInput.Update(msg)
			return m, cmd
		}

		return m.handleKey(msg)

	case tickMsg:
		return m.handleTick(msg)

	case systeminfoview.Msg:
		var cmd tea.Cmd
		m.systemInfo, cmd = m.systemInfo.Update(msg)
		return m, cmd

	default:
		return m.delegateToCurrentView(msg)
	}
}

func (m *Model) delegateToCurrentView(msg tea.Msg) (*Model, tea.Cmd) {
	var cmd tea.Cmd
	m.currentView, cmd = m.currentView.Update(msg)

	var vpCmd tea.Cmd
	m, vpCmd = m.updateViewports(msg)

	return m, tea.Batch(cmd, vpCmd)
}

func (m *Model) updateForResize(msg tea.WindowSizeMsg) (*Model, tea.Cmd) {
	var cmd tea.Cmd
	usableWidth := msg.Width - 4
	usableHeight := msg.Height - 10

	m.viewport.Width = usableWidth
	m.viewport.Height = usableHeight

	m.currentView, cmd = handleViewResize(m.currentView, usableWidth, usableHeight)
	return m, cmd
}

func handleViewResize(view view.View, width, height int) (*Model, tea.Cmd) {
	var adjustedMsg = tea.WindowSizeMsg{
		Width:  width,
		Height: height - systeminfoview.Height,
	}

	var cmd tea.Cmd
	view, cmd = view.Update(adjustedMsg)
	return view, cmd
}

func (m *Model) updateViewports(msg tea.Msg) (*Model, tea.Cmd) {
	var cmd1 tea.Cmd
	m.viewport, cmd1 = m.viewport.Update(msg)
	return m, cmd1
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global escape / quit handler
	if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEsc || msg.String() == "q" {
		var cmd tea.Cmd
		m, cmd = m.goBack()
		return m, cmd
	}

	var cmd tea.Cmd
	m.currentView, cmd = m.currentView.Update(msg)
	return m, cmd
}

func (m *Model) goBack() (*Model, tea.Cmd) {
	// If no parent view exists → quit the app
	if m.viewStack.Len() == 0 {
		exitCmd := m.currentView.OnExit()
		return m, tea.Batch(exitCmd, tea.Quit)
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
	m.currentView, resizeCmd = handleViewResize(m.currentView, m.viewport.Width, m.viewport.Height)

	// Execute all lifecycle commands
	return m, tea.Batch(exitCmd, enterCmd, resizeCmd)
}
