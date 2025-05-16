package main

import (
	tea "github.com/charmbracelet/bubbletea"
	"strings"
)

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.viewport.Width = msg.Width - 4
		m.viewport.Height = msg.Height - 10

		m.inspectViewport.Width = msg.Width - 4
		m.inspectViewport.Height = msg.Height - 10

	case tea.KeyMsg:
		switch msg.String() {
		case "esc", "q":
			switch {
			case m.inspecting:
				m.inspecting = false
				m.inspectText = ""
			case m.view == "nodeStacks":
				m.view = "main"
				m.nodeStackOutput = ""
			default:
				return m, tea.Quit
			}
			return m, nil
		}
		if m.commandMode {
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
				return m, nil
			case tea.KeyBackspace:
				if len(m.commandInput) > 0 {
					m.commandInput = m.commandInput[:len(m.commandInput)-1]
				}
				return m, nil
			default:
				m.commandInput += msg.String()
				return m, nil
			}
		} else if m.inspecting {
			if m.inspectSearchMode {
				// Search input mode inside inspect view
				switch msg.Type {
				case tea.KeyEnter:
					m.inspectSearchMode = false
					// Update highlighted content
					m.inspectViewport.SetContent(highlightMatches(m.inspectText, m.inspectSearchTerm))
					m.inspectViewport.GotoTop()
					return m, nil
				case tea.KeyEsc:
					m.inspectSearchMode = false
					m.inspectSearchTerm = ""
					m.inspectViewport.SetContent(m.inspectText) // reset content
					return m, nil
				case tea.KeyBackspace:
					if len(m.inspectSearchTerm) > 0 {
						m.inspectSearchTerm = m.inspectSearchTerm[:len(m.inspectSearchTerm)-1]
					}
					return m, nil
				default:
					// Append character to search term (handle printable characters)
					if len(msg.String()) == 1 && msg.String()[0] >= 32 {
						m.inspectSearchTerm += msg.String()
					}
					return m, nil
				}
			}
			// Scroll inside inspect viewport
			switch msg.String() {
			case "/":
				m.inspectSearchMode = true
				m.inspectSearchTerm = ""
				return m, nil
			case "j", "down":
				m.inspectViewport.LineDown(1)
			case "k", "up":
				m.inspectViewport.LineUp(1)
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
		} else {
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
					// We clear viewport content here, will fill on inspectMsg
					m.inspectViewport.SetContent("")
					return m, cmd
				}
			case ":":
				m.commandMode = true
			case "s":
				if m.mode == modeNodes && m.cursor < len(m.items) {
					fields := strings.Fields(m.items[m.cursor])
					if len(fields) > 0 {
						nodeID := fields[0]
						m.selectedNodeID = nodeID
						m.view = "nodeStacks"
						return m, loadNodeStacks(nodeID)
					}
				}

			}
		}

	case tickMsg:
		// Refresh both data and status every 5s
		return m, tea.Batch(loadData(m.mode), loadStatus())

	case loadedMsg:
		m.items = msg
		m.cursor = 0
		return m, nil

	case inspectMsg:
		m.inspecting = true
		m.inspectText = string(msg)
		m.inspectViewport.SetContent(m.inspectText)
		m.inspectViewport.GotoTop()
		return m, nil

	case nodeStackMsg:
		m.nodeStackOutput = string(msg)
		return m, nil

	case statusMsg:
		m.cpuUsage = msg.cpu
		m.memUsage = msg.mem
		m.containerCount = msg.containers
		m.serviceCount = msg.services
		return m, nil

	}

	// Update viewport and inspectViewport independently
	var cmd1, cmd2 tea.Cmd
	m.viewport, cmd1 = m.viewport.Update(msg)
	m.inspectViewport, cmd2 = m.inspectViewport.Update(msg)

	return m, tea.Batch(cmd1, cmd2)
}
