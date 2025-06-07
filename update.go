package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	inspectview "swarmcli/views/inspect"
	"swarmcli/views/logs"
	"swarmcli/views/stacks"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		var wndCmds []tea.Cmd
		m, wndCmds = m.handleResize(msg)
		cmds = append(cmds, wndCmds...)
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tickMsg:
		return m, tea.Batch(loadData(m.mode), loadStatus())
	case loadedMsg:
		m.items = msg
		m.cursor = 0
	case inspectview.Msg:
		m.view = inspectview.ViewName
		var cmd tea.Cmd
		m.inspect, cmd = m.inspect.Update(msg)
		return m, cmd
	case stacksview.Msg:
		m.view = stacksview.ViewName
		var cmd tea.Cmd
		m.stacks, cmd = m.stacks.Update(msg)
		return m, cmd
	case logs.Msg:
		m.view = logs.ViewName
		var cmd tea.Cmd
		m.logs, cmd = m.logs.Update(msg)
		return m, cmd
	case statusMsg:
		m.host = msg.host
		m.version = msg.version
		m.cpuUsage = msg.cpu
		m.memUsage = msg.mem
		m.containerCount = msg.containers
		m.serviceCount = msg.services
	default:
		var cmd tea.Cmd
		m.logs, cmd = m.logs.Update(msg)
		cmds = append(cmds, cmd)

		m.inspect, cmd = m.inspect.Update(msg)
		cmds = append(cmds, cmd)

		m.stacks, cmd = m.stacks.Update(msg)
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
	adjustedMsg := tea.WindowSizeMsg{
		Width:  usableWidth,
		Height: usableHeight,
	}

	m.inspect, cmd = m.inspect.Update(adjustedMsg)
	cmds = append(cmds, cmd)

	m.logs, cmd = m.logs.Update(adjustedMsg)
	cmds = append(cmds, cmd)

	m.stacks, cmd = m.stacks.Update(adjustedMsg)
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
	if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEsc || msg.String() == "esc" {
		switch {
		case m.view == inspectview.ViewName:
			m.view = "main"
			var cmd tea.Cmd
			m.inspect, cmd = m.inspect.Update(msg)
			return m, cmd
		case m.view == stacksview.ViewName:
			m.view = "main"
			var cmd tea.Cmd
			m.stacks, cmd = m.stacks.Update(msg)
			return m, cmd
		case m.view == logs.ViewName:
			m.view = stacksview.ViewName
			var cmd tea.Cmd
			m.logs, cmd = m.logs.Update(msg)
			return m, cmd
		default:
			return m, tea.Quit
		}
	}

	if m.commandMode {
		return m.handleCommandKey(msg)
	} else {
		switch m.view {
		case inspectview.ViewName:
			var cmd tea.Cmd
			m.inspect, cmd = m.inspect.Update(msg)
			return m, cmd
		case logs.ViewName:
			var cmd tea.Cmd
			m.logs, cmd = m.logs.Update(msg)
			return m, cmd
		case stacksview.ViewName:
			var cmd tea.Cmd
			m.stacks, cmd = m.stacks.Update(msg)
			return m, cmd
		case "main":
			return m.handleMainKey(msg)
		default:
			return m.handleMainKey(msg)
		}
	}
}

func (m model) handleCommandKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		cmd := strings.TrimSpace(m.commandInput)
		m.commandMode = false
		m.commandInput = ""
		switch cmd {
		case "nodes":
			m.mode = modeNodes
			m.cursor = 0
			return m, loadData(modeNodes)
		case "services":
			m.mode = modeServices
			m.cursor = 0
			return m, loadData(modeServices)
		case "stacks":
			m.mode = modeStacks
			m.cursor = 0
			return m, loadData(modeStacks)
		}
	case tea.KeyEsc:
		m.commandMode = false
		m.commandInput = ""
	case tea.KeyBackspace:
		if len(m.commandInput) > 0 {
			m.commandInput = m.commandInput[:len(m.commandInput)-1]
		}
	default:
		m.commandInput += msg.String()
	}
	return m, nil
}

func (m model) handleMainKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "j", "down":
		if m.cursor < len(m.items)-1 {
			m.cursor++
		}

	case "k", "up":
		if m.cursor > 0 {
			m.cursor--
		}

	case "i":
		if m.cursor < len(m.items) {
			cmd := inspectItem(m.mode, m.items[m.cursor])
			m.inspect.SetContent("")
			return m, cmd
		}

	case ":":
		m.commandMode = true

	case "s":
		return m.handleSelectNode()

	}
	return m, nil
}

func (m model) handleSelectNode() (tea.Model, tea.Cmd) {
	if m.mode != modeNodes || m.cursor >= len(m.items) {
		return m, nil
	}

	fields := strings.Fields(m.items[m.cursor])
	if len(fields) == 0 {
		return m, nil
	}

	nodeID := fields[0]
	m.selectedNodeID = nodeID
	return m, stacksview.LoadNodeStacks(nodeID)
}
