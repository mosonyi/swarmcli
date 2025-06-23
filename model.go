package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"swarmcli/styles"
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
	mode        mode
	view        string // "main" or "nodeStacks"
	viewport    viewport.Model
	initialized bool
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

	return model{
		mode:      modeNodes,
		viewport:  vp,
		viewStack: []view.View{},
	}
}

func (m model) switchToView(name string, data any) (model, tea.Cmd) {
	var newView view.View
	var loadCmd tea.Cmd

	switch name {
	case logsview.ViewName:
		serviceID := data.(string)
		newView = logsview.New(m.viewport.Width, m.viewport.Height)
		loadCmd = logsview.Load(serviceID)

	case stacksview.ViewName:
		nodeID := data.(string)
		newView = stacksview.New(m.viewport.Width, m.viewport.Height)
		loadCmd = stacksview.LoadNodeStacks(nodeID)

	case inspectview.ViewName:
		nodeViewLine := data.(string)
		newView = inspectview.New(m.viewport.Width, m.viewport.Height)
		loadCmd = inspectview.LoadInspectItem(nodeViewLine)
	default:
		return m, nil
	}

	// Push old view to stack
	m.viewStack = append(m.viewStack, m.currentView)
	m.view = name

	newView, resizeCmd := handleViewResize(newView, m.viewport.Width, m.viewport.Height)
	m.currentView = newView

	return m, tea.Batch(resizeCmd, loadCmd)
}

func (m model) renderStackBar() string {
	// Combine stack and current view
	stack := append(m.viewStack, m.currentView)

	var parts []string
	for i, view := range stack {
		if i > 0 {
			parts = append(parts, lipgloss.NewStyle().Faint(true).Render(" → "))

		}
		style := styles.Rainbow[i%len(styles.Rainbow)]
		label := view.Name()
		parts = append(parts, style.Render(fmt.Sprintf(" %s ", label)))
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}
