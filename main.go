package main

import (
	"log"
	inspectview "swarmcli/views/inspect"
	logsview "swarmcli/views/logs"
	nodesview "swarmcli/views/nodes"
	stacksview "swarmcli/views/stacks"
	systeminfoview "swarmcli/views/systeminfo"
	"swarmcli/views/view"
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

var viewRegistry = map[string]view.Factory{}

func registerView(name string, factory view.Factory) {
	viewRegistry[name] = factory
}

func init() {
	registerView(logsview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		return logsview.New(w, h), logsview.Load(payload.(string))
	})
	registerView(inspectview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		return inspectview.New(w, h), inspectview.LoadInspectItem(payload.(string))
	})
	registerView(stacksview.ViewName, func(w, h int, payload any) (view.View, tea.Cmd) {
		return stacksview.New(w, h), stacksview.LoadNodeStacks(payload.(string))
	})
}

func main() {
	p := tea.NewProgram(initialModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatal(err)
	}
}
