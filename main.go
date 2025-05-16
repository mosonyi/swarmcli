package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"swarmcli/docker"

	tea "github.com/charmbracelet/bubbletea"
)

type mode string

const (
	modeNodes    mode = "nodes"
	modeServices mode = "services"
	modeStacks   mode = "stacks"
	version           = "dev"
)

type model struct {
	mode            mode
	view            string // "main" or "nodeStacks"
	items           []string
	cursor          int
	inspecting      bool
	inspectText     string
	commandMode     bool
	commandInput    string
	selectedNodeID  string
	nodeStackOutput string

	// status overview fields
	cpuUsage       string
	memUsage       string
	containerCount int
	serviceCount   int
}

// initialModel creates default model
func initialModel() model {
	return model{mode: modeNodes}
}

// ------- Messages

type tickMsg time.Time
type loadedMsg []string
type inspectMsg string
type nodeStackMsg string

// new status message to update system usage info
type statusMsg struct {
	cpu        string
	mem        string
	containers int
	services   int
}

// ------- Update

func (m model) Init() tea.Cmd {
	return tea.Batch(tick(), loadData(m.mode), loadStatus())
}

func tick() tea.Cmd {
	return tea.Tick(time.Second*5, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func loadData(m mode) tea.Cmd {
	return func() tea.Msg {
		var list []string
		switch m {
		case modeNodes:
			nodes, _ := docker.ListSwarmNodes()
			for _, n := range nodes {
				list = append(list, fmt.Sprint(n))
			}
		case modeServices:
			services, _ := docker.ListSwarmServices()
			for _, s := range services {
				list = append(list, fmt.Sprint(s))
			}
		case modeStacks:
			stacks, _ := docker.ListStacks()
			for _, s := range stacks {
				list = append(list, fmt.Sprint(s))
			}
		}
		return loadedMsg(list)
	}
}

func loadStatus() tea.Cmd {
	return func() tea.Msg {
		// Use your docker package functions here
		cpu := docker.GetSwarmCPUUsage()
		mem := docker.GetSwarmMemUsage()
		containers := docker.GetContainerCount()
		services := docker.GetServiceCount()

		return statusMsg{
			cpu:        cpu,
			mem:        mem,
			containers: containers,
			services:   services,
		}
	}
}

func inspectItem(mode mode, line string) tea.Cmd {
	return func() tea.Msg {
		item := strings.Fields(line)[0]
		var out []byte
		var err error
		switch mode {
		case modeNodes:
			out, err = exec.Command("docker", "node", "inspect", item).CombinedOutput()
		case modeServices:
			out, err = exec.Command("docker", "service", "inspect", item).CombinedOutput()
		case modeStacks:
			out, err = exec.Command("docker", "stack", "services", item).CombinedOutput()
		}
		if err != nil {
			return inspectMsg(fmt.Sprintf("Error: %v\n%s", err, out))
		}
		return inspectMsg(string(out))
	}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

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
					return m, inspectItem(m.mode, m.items[m.cursor])
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

	return m, nil
}

// ------- View

const (
	ColorYellow = "\033[33m"
	ColorWhite  = "\033[37m"
	ColorReset  = "\033[0m"
)

func (m model) View() string {
	if m.inspecting {
		return fmt.Sprintf("Inspecting (%s)\n\n%s\n[press q or esc to go back]", m.mode, m.inspectText)
	}

	if m.view == "nodeStacks" {
		return fmt.Sprintf(
			"Stacks on node: %s\n\n%s\n\n[press q or esc to go back]",
			m.selectedNodeID, m.nodeStackOutput)
	}

	s := fmt.Sprintf(
		"%sCPU: %-10s  MEM: %-10s  Containers: %-4d  Services: %-4d%s\n\n",
		ColorYellow, m.cpuUsage, m.memUsage, m.containerCount, m.serviceCount, ColorReset,
	)

	s += fmt.Sprintf("Mode: %s (press : to switch)\n\n", m.mode)
	for i, item := range m.items {
		cursor := "  "
		if i == m.cursor {
			cursor = "→ "
		}
		s += fmt.Sprintf("%s%s\n", cursor, item)
	}
	if m.commandMode {
		s += fmt.Sprintf("\n: %s", m.commandInput)
	} else {
		s += "\n[i: inspect, s: see stacks q: quit, j/k: move]"
	}
	return s
}

// ------- Node stacks view

func loadNodeStacks(nodeID string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("node=%s", nodeID), "--format", "table {{.Names}}\t{{.Image}}\t{{.Status}}\t{{.Labels}}")
		out, err := cmd.CombinedOutput()
		if err != nil {
			return nodeStackMsg(fmt.Sprintf("Error: %v\n%s", err, out))
		}
		return nodeStackMsg(string(out))
	}
}

// ------- Main

func main() {
	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
