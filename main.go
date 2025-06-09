package main

import (
	"log"
	nodesview "swarmcli/views/nodes"
	systeminfoview "swarmcli/views/systeminfo"
	"time"

	tea "github.com/charmbracelet/bubbletea"
)

const (
	modeNodes mode   = "nodes"
	version   string = "dev"
)

// ------- Messages

type tickMsg time.Time

func (m model) Init() tea.Cmd {
	return tea.Batch(tick(), nodesview.LoadNodes(), systeminfoview.LoadStatus())
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
