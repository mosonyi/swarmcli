package main

import (
	tea "github.com/charmbracelet/bubbletea"
	inspectview "swarmcli/views/inspect"
	"swarmcli/views/logs"
	nodesview "swarmcli/views/nodes"
	"swarmcli/views/stacks"
	systeminfoview "swarmcli/views/systeminfo"
	"swarmcli/views/view"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case view.NavigateToMsg:
		return m.switchToView(msg.ViewName, msg.Payload)
	case tea.WindowSizeMsg:
		var wndCmds []tea.Cmd
		m, wndCmds = m.handleResize(msg)
		cmds = append(cmds, wndCmds...)
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tickMsg:
		return m, tea.Batch(nodesview.LoadNodes(), systeminfoview.LoadStatus())
	case systeminfoview.Msg:
		var cmd tea.Cmd
		m.systemInfo, cmd = m.systemInfo.Update(msg)
		return m, cmd
	default:
		var cmd tea.Cmd
		m.currentView, cmd = m.currentView.Update(msg)
		cmds = append(cmds, cmd)
	}

	var cmd tea.Cmd
	m, cmd = m.updateViewports(msg)
	cmds = append(cmds, cmd)
	return m, tea.Batch(cmds...)
}

func (m model) handleResize(msg tea.WindowSizeMsg) (model, []tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	usableWidth := msg.Width - 4
	usableHeight := msg.Height - 10

	m.viewport.Width = usableWidth
	m.viewport.Height = usableHeight

	// Create adjusted WindowSizeMsg
	var adjustedMsg tea.WindowSizeMsg

	if m.currentView.Name() == nodesview.ViewName {
		adjustedMsg = tea.WindowSizeMsg{
			Width:  usableWidth,
			Height: usableHeight / 2,
		}
	} else {
		adjustedMsg = tea.WindowSizeMsg{
			Width:  usableWidth,
			Height: usableHeight,
		}
	}

	m.currentView, cmd = m.currentView.Update(adjustedMsg)
	cmds = append(cmds, cmd)

	return m, cmds
}

func (m model) updateViewports(msg tea.Msg) (model, tea.Cmd) {
	var cmd1 tea.Cmd
	m.viewport, cmd1 = m.viewport.Update(msg)
	return m, cmd1
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global escape / quit handler
	if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEsc || msg.String() == "q" {
		switch {
		case m.view == inspectview.ViewName:
			m.view = "main"
			var cmd tea.Cmd
			m.currentView, cmd = m.currentView.Update(msg)
			return m, cmd
		case m.view == stacksview.ViewName:
			m.view = "main"
			var cmd tea.Cmd
			m.currentView, cmd = m.currentView.Update(msg)
			return m, cmd
		case m.view == logsview.ViewName:
			m.view = stacksview.ViewName
			var cmd tea.Cmd
			m.currentView, cmd = m.currentView.Update(msg)
			return m, cmd
		default:
			return m, tea.Quit
		}
	}

	//if m.commandMode {
	//	//return m.handleCommandKey(msg)
	//} else {
	switch m.view {
	case inspectview.ViewName:
		var cmd tea.Cmd
		m.currentView, cmd = m.currentView.Update(msg)
		return m, cmd
	case logsview.ViewName:
		var cmd tea.Cmd
		m.currentView, cmd = m.currentView.Update(msg)
		return m, cmd
	case stacksview.ViewName:
		var cmd tea.Cmd
		m.currentView, cmd = m.currentView.Update(msg)
		return m, cmd
	//case ":":
	//	m.commandMode = true
	case "main":
		var cmd tea.Cmd
		m.currentView, cmd = m.currentView.Update(msg)
		return m, cmd

	default:
		var cmd tea.Cmd
		m.currentView, cmd = m.currentView.Update(msg)
		return m, cmd
	}
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
