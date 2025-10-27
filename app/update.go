package app

import (
	"strings"
	"swarmcli/commands"
	"swarmcli/commands/api"
	"swarmcli/views/commandinput"
	systeminfoview "swarmcli/views/systeminfo"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case commandinput.SubmitMsg:
		cmdName := strings.TrimSpace(msg.Command)
		cmd, ok := commands.Get(cmdName)
		if !ok {
			m.commandInput.ShowError("Unknown command: " + cmdName)
			return m, nil
		}

		ctx := api.Context{App: &m}
		return m, cmd.Execute(ctx, nil)

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

func (m Model) delegateToCurrentView(msg tea.Msg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	m.currentView, cmd = m.currentView.Update(msg)

	var vpCmd tea.Cmd
	m, vpCmd = m.updateViewports(msg)

	return m, tea.Batch(cmd, vpCmd)
}

func (m Model) updateForResize(msg tea.WindowSizeMsg) (Model, tea.Cmd) {
	var cmd tea.Cmd
	usableWidth := msg.Width - 4
	usableHeight := msg.Height - 10

	m.viewport.Width = usableWidth
	m.viewport.Height = usableHeight

	m.currentView, cmd = handleViewResize(m.currentView, usableWidth, usableHeight)
	return m, cmd
}

func handleViewResize(view view.View, width, height int) (view.View, tea.Cmd) {
	var adjustedMsg = tea.WindowSizeMsg{
		Width:  width,
		Height: height - systeminfoview.Height,
	}

	var cmd tea.Cmd
	view, cmd = view.Update(adjustedMsg)
	return view, cmd
}

func (m Model) updateViewports(msg tea.Msg) (Model, tea.Cmd) {
	var cmd1 tea.Cmd
	m.viewport, cmd1 = m.viewport.Update(msg)
	return m, cmd1
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (m Model) goBack() (Model, tea.Cmd) {
	if m.viewStack.Len() == 0 {
		return m, tea.Quit
	}
	m.currentView = m.viewStack.Pop()
	return m, nil
}

// 🟢 Add this function to dispatch commands (simplified for now)
func (m Model) executeCommand(line string) (tea.Model, tea.Cmd) {
	fields := strings.Fields(line)
	if len(fields) == 0 {
		return m, nil
	}

	switch {
	case len(fields) >= 3 && fields[0] == "docker" && fields[1] == "stack" && fields[2] == "ls":
		return m.switchToView("stacks", nil)

	case len(fields) >= 3 && fields[0] == "docker" && fields[1] == "service" && fields[2] == "ls":
		return m.switchToView("services", nil)

	default:
		return m, tea.Printf("Unknown command: %s", line)
	}
}
