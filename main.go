package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"
	"time"

	"swarmcli/docker"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type mode string

const (
	modeNodes    mode = "nodes"
	modeServices mode = "services"
	modeStacks   mode = "stacks"
	version           = "dev"
)

// Styles with lipgloss

var (
	borderStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#874BFD")).
			Padding(0, 1)

	statusStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#00FF00")).
			Padding(0, 1).
			Width(50)

	listStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("#FFD700")).
			Margin(1, 0).
			Padding(1)

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#888888")).
			Italic(true).
			Margin(1, 0)
)

// Model holds app state
type model struct {
	mode            mode
	view            string // "main" or "nodeStacks"
	items           []string
	cursor          int
	viewport        viewport.Model
	inspectViewport viewport.Model
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

	// Search inside inspect view
	inspectSearchMode bool   // Are we in search mode inside inspect view?
	inspectSearchTerm string // The search term to highlight
}

// initialModel creates default model
func initialModel() model {
	vp := viewport.New(80, 20)
	vp.YPosition = 5

	inspectVp := viewport.New(80, 20) // initial size, will update on WindowSizeMsg

	return model{
		mode:            modeNodes,
		viewport:        vp,
		inspectViewport: inspectVp,
	}
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

func highlightMatches(text, searchTerm string) string {
	if searchTerm == "" {
		return text
	}
	re, err := regexp.Compile("(?i)" + regexp.QuoteMeta(searchTerm)) // case-insensitive
	if err != nil {
		return text // fail silently
	}
	highlighted := re.ReplaceAllStringFunc(text, func(match string) string {
		return lipgloss.NewStyle().Background(lipgloss.Color("238")).Foreground(lipgloss.Color("229")).Render(match)
	})
	return highlighted
}

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

// ------- View

func (m model) View() string {
	if m.inspecting {
		header := fmt.Sprintf("Inspecting (%s)", m.mode)
		if m.inspectSearchMode {
			header += fmt.Sprintf(" - Search: %s", m.inspectSearchTerm)
		}
		return borderStyle.Render(
			fmt.Sprintf("%s\n\n%s\n\n[press q or esc to go back, / to search]", header, m.inspectViewport.View()),
		)
	}

	if m.view == "nodeStacks" {
		return borderStyle.Render(
			fmt.Sprintf("Stacks on node: %s\n\n%s\n\n[press q or esc to go back]", m.selectedNodeID, m.nodeStackOutput),
		)
	}

	status := statusStyle.Render(fmt.Sprintf(
		"CPU: %s\nMEM: %s\nContainers: %d\nServices: %d",
		m.cpuUsage, m.memUsage, m.containerCount, m.serviceCount,
	))

	helpText := helpStyle.Render("[i: inspect, s: see stacks, q: quit, j/k: move cursor, : switch mode]")

	// Show the main list with cursor highlighted, no viewport scroll for this version
	s := fmt.Sprintf("Mode: %s\n\n", m.mode)
	for i, item := range m.items {
		cursor := "  "
		if i == m.cursor {
			cursor = "→ "
		}
		s += fmt.Sprintf("%s%s\n", cursor, item)
	}

	return lipgloss.JoinVertical(
		lipgloss.Left,
		status,
		borderStyle.Render(s),
		helpText,
	)
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
