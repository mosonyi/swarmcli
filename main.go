package main

import (
	"log"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

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
type logMsg string
type nodeStacksMsg struct {
	output   string
	stacks   []string
	services []string
}

// new status message to update system usage info
type statusMsg struct {
	host       string
	version    string
	cpu        string
	mem        string
	containers int
	services   int
}

func (m model) Init() tea.Cmd {
	return tea.Batch(tick(), loadData(m.mode), loadStatus())
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
