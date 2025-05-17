package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
	"swarmcli/views/logs"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m = m.handleResize(msg)
	case tea.KeyMsg:
		return m.handleKey(msg)
	case tickMsg:
		return m, tea.Batch(loadData(m.mode), loadStatus())
	case loadedMsg:
		m.items = msg
		m.cursor = 0
	case inspectMsg:
		m.inspecting = true
		m.inspectText = string(msg)
		m.inspectViewport.SetContent(m.inspectText)
		m.inspectViewport.GotoTop()
	case nodeStacksMsg:
		m.view = "nodeStacks"
		m.nodeStacks = msg.stacks
		m.nodeServices = msg.services
		m.nodeStackLines = strings.Split(msg.output, "\n")
		m.stackCursor = 0
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
	}

	return m.updateViewports(msg)
}

func (m model) handleResize(msg tea.WindowSizeMsg) model {
	usableWidth := msg.Width - 4
	usableHeight := msg.Height - 10

	m.viewport.Width = usableWidth
	m.viewport.Height = usableHeight

	m.inspectViewport.Width = usableWidth
	m.inspectViewport.Height = usableHeight

	m.logs = m.logs.SetSize(usableWidth, usableHeight)

	return m
}

func (m model) updateViewports(msg tea.Msg) (model, tea.Cmd) {
	var cmd1, cmd2 tea.Cmd
	m.viewport, cmd1 = m.viewport.Update(msg)
	m.inspectViewport, cmd2 = m.inspectViewport.Update(msg)
	return m, tea.Batch(cmd1, cmd2)
}

func (m model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Global escape / quit handler
	if msg.Type == tea.KeyCtrlC || msg.Type == tea.KeyEsc || msg.String() == "esc" {
		switch {
		case m.inspecting:
			m.inspecting = false
			m.inspectText = ""
			return m, nil
		case m.view == "nodeStacks":
			m.view = "main"
			return m, nil
		case m.view == logs.ViewName:
			m.view = "nodeStacks"
			var cmd tea.Cmd
			m.logs, cmd = m.logs.Update(msg)
			return m, cmd
		default:
			return m, tea.Quit
		}
	}

	if m.commandMode {
		return m.handleCommandKey(msg)
	} else if m.inspecting {
		return m.handleInspectKey(msg)
	} else {
		switch m.view {
		case logs.ViewName:
			var cmd tea.Cmd
			m.logs, cmd = m.logs.Update(msg)
			return m, cmd
		case "nodeStacks", "main":
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

func (m model) handleInspectKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case m.inspectSearchMode:
		return m.handleInspectSearchKey(msg)
	default:
		return m.handleInspectScrollKey(msg)
	}
}

func (m model) handleInspectSearchKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Type {
	case tea.KeyEnter:
		m.inspectSearchMode = false
		m.inspectViewport.SetContent(highlightMatches(m.inspectText, m.inspectSearchTerm))
		m.inspectViewport.GotoTop()
	case tea.KeyEsc:
		m.inspectSearchMode = false
		m.inspectSearchTerm = ""
		m.inspectViewport.SetContent(m.inspectText)
	case tea.KeyBackspace:
		if len(m.inspectSearchTerm) > 0 {
			m.inspectSearchTerm = m.inspectSearchTerm[:len(m.inspectSearchTerm)-1]
		}
	default:
		if len(msg.String()) == 1 && msg.String()[0] >= 32 {
			m.inspectSearchTerm += msg.String()
		}
	}
	return m, nil
}

func (m model) handleInspectScrollKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "/":
		m.inspectSearchMode = true
		m.inspectSearchTerm = ""
	case "j", "down":
		m.inspectViewport.ScrollDown(1)
	case "k", "up":
		m.inspectViewport.ScrollUp(1)
	case "pgdown":
		m.inspectViewport.ScrollDown(m.inspectViewport.Height)
	case "pgup":
		m.inspectViewport.ScrollUp(m.inspectViewport.Height)
	case "g":
		m.inspectViewport.GotoTop()
	case "G":
		m.inspectViewport.GotoBottom()
	case "q", "esc":
		m.inspecting = false
		m.inspectText = ""
	}
	return m, nil
}

func (m model) handleMainKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q":
		return m, tea.Quit

	case "j", "down":
		if m.view == "nodeStacks" && m.stackCursor < len(m.nodeStacks)-1 {
			m.stackCursor++
		} else if m.cursor < len(m.items)-1 {
			m.cursor++
		}

	case "k", "up":
		if m.view == "nodeStacks" && m.stackCursor > 0 {
			m.stackCursor--
		} else if m.cursor > 0 {
			m.cursor--
		}

	case "enter":
		if m.view == "nodeStacks" && m.stackCursor < len(m.nodeStackLines) {
			serviceID := m.nodeServices[m.stackCursor]
			return m, logs.Load(serviceID)
		}

	case "i":
		if m.cursor < len(m.items) {
			cmd := inspectItem(m.mode, m.items[m.cursor])
			m.inspectViewport.SetContent("")
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
	m.view = "nodeStacks"
	return m, loadNodeStacks(nodeID)
}
