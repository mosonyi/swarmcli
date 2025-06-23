package main

import (
	"fmt"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"swarmcli/styles"
	nodesview "swarmcli/views/nodes"
	systeminfoview "swarmcli/views/systeminfo"
	"swarmcli/views/view"
	"swarmcli/views/viewstack"
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

	currentView view.View
	viewStack   viewstack.Stack
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
		viewStack:   viewstack.Stack{},
	}
}

func (m model) switchToView(name string, data any) (model, tea.Cmd) {
	factory, ok := viewRegistry[name]
	if !ok {
		return m, nil
	}

	newView, loadCmd := factory(m.viewport.Width, m.viewport.Height, data)
	newView, resizeCmd := handleViewResize(newView, m.viewport.Width, m.viewport.Height)

	m.viewStack.Push(m.currentView)
	m.currentView = newView
	m.view = name

	return m, tea.Batch(resizeCmd, loadCmd)
}

func (m model) renderStackBar() string {
	// Combine stack and current view
	stack := append(m.viewStack.Views(), m.currentView)

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
