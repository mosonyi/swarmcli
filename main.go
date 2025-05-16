package main

import (
	"fmt"
	"os"
	"os/exec"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

type mode string

const (
	modeNodes    mode = "nodes"
	modeServices mode = "services"
	modeStacks   mode = "stacks"
	version           = "dev"
)

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
