package main

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	inspectview "swarmcli/views/inspect"
	"swarmcli/views/logs"
	nodesview "swarmcli/views/nodes"
	stacksview "swarmcli/views/stacks"
	systeminfoview "swarmcli/views/systeminfo"
	"swarmcli/views/view"
)

type mode string

// Model holds app state
type model struct {
	mode     mode
	view     string // "main" or "nodeStacks"
	viewport viewport.Model
	//commandMode  bool
	//commandInput string

	systemInfo systeminfoview.Model
	nodes      nodesview.Model
	stacks     stacksview.Model
	logs       logsview.Model
	inspect    inspectview.Model

	currentView view.View
	views       map[string]view.View
	viewStack   []view.View
}

// initialModel creates default model
func initialModel() model {
	vp := viewport.New(80, 20)
	vp.YPosition = 5

	nodes := nodesview.New(80, 20)

	return model{
		mode:        modeNodes,
		viewport:    vp,
		currentView: nodes,
		viewStack:   []view.View{&nodes},
	}
}

func (m model) switchToView(name string, data any) (model, tea.Cmd) {
	switch name {
	case logsview.ViewName:
		serviceID := data.(string)
		logsView := logsview.New(80, 20)
		m.viewStack = append(m.viewStack, m.currentView)
		m.currentView = logsView
		return m, logsview.Load(serviceID)

	case stacksview.ViewName:
		nodeID := data.(string)
		stacksView := stacksview.New(80, 20)
		m.viewStack = append(m.viewStack, m.currentView)
		m.currentView = stacksView
		return m, stacksview.LoadNodeStacks(nodeID)

	case inspectview.ViewName:
		nodeViewLine := data.(string)
		stacksView := inspectview.New(80, 20)
		m.viewStack = append(m.viewStack, m.currentView)
		m.currentView = stacksView
		return m, inspectview.LoadInspectItem(nodeViewLine)
	}

	return m, nil
}
