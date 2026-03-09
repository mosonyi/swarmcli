// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package app

import (
	"fmt"
	"swarmcli/docker"
	"swarmcli/ui"
	"swarmcli/views/commandinput"
	loadingview "swarmcli/views/loading"
	"swarmcli/views/searchinput"
	systeminfoview "swarmcli/views/systeminfo"
	"swarmcli/views/view"
	"swarmcli/views/viewstack"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model holds app state
type Model struct {
	deps docker.Deps

	viewport viewport.Model

	systemInfo *systeminfoview.Model

	currentView view.View
	viewStack   viewstack.Stack

	commandInput *commandinput.Model
	searchInput  *searchinput.Model

	// Terminal dimensions
	terminalWidth  int
	terminalHeight int

	// Fullscreen mode — hides helpbar/stackbar, uses full terminal
	fullscreen bool
}

var depsTransform func(docker.Deps) docker.Deps

// SetDepsTransform registers a function that transforms the default Deps
// before they are stored in the Model. Must be called before InitialModel().
// Used by extension builds to wrap Deps interfaces with middleware.
func SetDepsTransform(fn func(docker.Deps) docker.Deps) {
	depsTransform = fn
}

func buildDeps() docker.Deps {
	deps := docker.DefaultDeps()
	if depsTransform != nil {
		deps = depsTransform(deps)
	}
	return deps
}

func InitialModel() *Model {
	// Use larger initial dimensions that will be adjusted by first WindowSizeMsg
	// This avoids the loading view appearing too small on the first render
	terminalWidth, terminalHeight := 200, 50

	vp := viewport.New(terminalWidth, terminalHeight)
	// Start viewport below the system info header (original default)
	vp.YPosition = 5

	loading := loadingview.New(terminalWidth, terminalHeight, true, map[string]string{
		"title":   "Initializing",
		"header":  "Fetching cluster info",
		"message": "Loading Swarm nodes and stacks...",
	})

	deps := buildDeps()

	return &Model{
		deps:           deps,
		viewport:       vp,
		currentView:    loading,
		systemInfo:     systeminfoview.New(deps, version),
		viewStack:      viewstack.Stack{},
		commandInput:   cmdBar(),
		searchInput:    searchinput.New(),
		terminalWidth:  terminalWidth,
		terminalHeight: terminalHeight,
	}
}

// Init  will be automatically called by Bubble Tea if the model implements the Model interface
// and is passed into the tea.NewProgram function.
func (m *Model) Init() tea.Cmd {
	// "" loads all stacks on all nodes
	return tea.Batch(
		tick(),
		loadSnapshotAsync(),
		m.systemInfo.LoadStatus(),
		m.systemInfo.Init(), // Initialize systeminfo's tick commands
		docker.WatchEventsCmd(),
	)
}

func (m *Model) switchToView(name string, data any) tea.Cmd {
	factory, ok := view.GetFactory(name)
	if !ok {
		return nil
	}

	// Exit hook for current view
	exitCmd := m.currentView.OnExit()

	newView, loadCmd := factory(m.deps, m.viewport.Width, m.viewport.Height, data)
	resizeCmd := handleViewResize(newView, m.viewport.Width, m.viewport.Height, false)

	// Push current view onto stack
	m.viewStack.Push(m.currentView)
	m.currentView = newView

	// Enter hook for new view
	enterCmd := newView.OnEnter()

	return tea.Batch(exitCmd, resizeCmd, loadCmd, enterCmd)
}

func (m *Model) replaceView(name string, data any) tea.Cmd {
	factory, ok := view.GetFactory(name)
	if !ok {
		return nil
	}

	// Run exit hook on current view
	exitCmd := m.currentView.OnExit()

	newView, loadCmd := factory(m.deps, m.viewport.Width, m.viewport.Height, data)
	resizeCmd := handleViewResize(newView, m.viewport.Width, m.viewport.Height, false)

	m.currentView = newView
	m.viewStack.Reset()

	// Run enter hook on new view
	enterCmd := newView.OnEnter()

	return tea.Batch(exitCmd, resizeCmd, loadCmd, enterCmd)
}

func (m *Model) renderStackBar() string {
	// Combine stack and current view
	stack := append(m.viewStack.Views(), m.currentView)

	var parts []string
	for i, v := range stack {
		if i > 0 {
			parts = append(parts, lipgloss.NewStyle().Faint(true).Render(" → "))

		}
		style := ui.Rainbow[i%len(ui.Rainbow)]
		label := v.Name()
		parts = append(parts, style.Render(fmt.Sprintf(" %s ", label)))
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}

func cmdBar() *commandinput.Model {
	cmdBar := commandinput.New()
	return cmdBar
}
