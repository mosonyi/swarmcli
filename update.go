package main

import (
	tea "github.com/charmbracelet/bubbletea"
	nodesview "swarmcli/views/nodes"
	systeminfoview "swarmcli/views/systeminfo"
	"swarmcli/views/view"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case view.NavigateToMsg:
		return m.switchToView(msg.ViewName, msg.Payload)

	case tea.WindowSizeMsg:
		return m.updateForResize(msg)

	case tea.KeyMsg:
		return m.handleKey(msg)

	case tickMsg:
		return m, tea.Batch(
			nodesview.LoadNodes(),
			systeminfoview.LoadStatus(),
		)

	case systeminfoview.Msg:
		var cmd tea.Cmd
		m.systemInfo, cmd = m.systemInfo.Update(msg)
		return m, cmd

	default:
		return m.delegateToCurrentView(msg)
	}
}

func (m model) delegateToCurrentView(msg tea.Msg) (model, tea.Cmd) {
	var cmd tea.Cmd
	m.currentView, cmd = m.currentView.Update(msg)

	var vpCmd tea.Cmd
	m, vpCmd = m.updateViewports(msg)

	return m, tea.Batch(cmd, vpCmd)
}

func (m model) updateForResize(msg tea.WindowSizeMsg) (model, tea.Cmd) {
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

func (m model) updateViewports(msg tea.Msg) (model, tea.Cmd) {
	var cmd1 tea.Cmd
	m.viewport, cmd1 = m.viewport.Update(msg)
	return m, cmd1
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

func (m model) goBack() (model, tea.Cmd) {
	if m.viewStack.Len() == 0 {
		return m, tea.Quit
	}
	m.currentView = m.viewStack.Pop()
	return m, nil
}

//func (m model) handleCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
//	switch msg.Type {
//	case tea.KeyEnter:
//		cmd := strings.TrimSpace(m.commandInput)
//		m.commandMode = false
//		m.commandInput = ""
//		switch cmd {
//		case "nodes":
//			m.mode = modeNodes
//			m.cursor = 0
//			return m, loadData(modeNodes)
//		case "services":
//			m.mode = modeServices
//			m.cursor = 0
//			return m, loadData(modeServices)
//		case "stacks":
//			m.mode = modeStacks
//			m.cursor = 0
//			return m, loadData(modeStacks)
//		}
//	case tea.KeyEsc:
//		m.commandMode = false
//		m.commandInput = ""
//	case tea.KeyBackspace:
//		if len(m.commandInput) > 0 {
//			m.commandInput = m.commandInput[:len(m.commandInput)-1]
//		}
//	default:
//		m.commandInput += msg.String()
//	}
//	return m, nil
//}
