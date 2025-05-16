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
	mode         mode
	items        []string
	cursor       int
	inspecting   bool
	inspectText  string
	commandMode  bool
	commandInput string
}

func initialModel() model {
	return model{mode: modeNodes}
}

// ------- Messages

type tickMsg time.Time
type loadedMsg []string
type inspectMsg string

// ------- Update

func (m model) Init() tea.Cmd {
	return tea.Batch(tick(), loadData(m.mode))
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
		if m.inspecting {
			switch msg.String() {
			case "esc", "q":
				m.inspecting = false
				m.inspectText = ""
				return m, nil
			}
		} else if m.commandMode {
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
			}
		}

	case tickMsg:
		return m, loadData(m.mode)

	case loadedMsg:
		m.items = msg
		m.cursor = 0
		return m, nil

	case inspectMsg:
		m.inspecting = true
		m.inspectText = string(msg)
		return m, nil
	}

	return m, nil
}

// ------- View

func (m model) View() string {
	if m.inspecting {
		return fmt.Sprintf("Inspecting (%s)\n\n%s\n[press q or esc to go back]", m.mode, m.inspectText)
	}

	s := fmt.Sprintf("Mode: %s (press : to switch)\n\n", m.mode)
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
		s += "\n[i: inspect, q: quit, j/k: move]"
	}
	return s
}

// ------- Main

func main() {
	p := tea.NewProgram(initialModel())
	if err := p.Start(); err != nil {
		fmt.Println("Error running program:", err)
		os.Exit(1)
	}
}
