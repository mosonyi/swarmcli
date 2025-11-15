package app

import (
	"fmt"
	"swarmcli/ui"
	"swarmcli/views/commandinput"
	loadingview "swarmcli/views/loading"
	systeminfoview "swarmcli/views/systeminfo"
	"swarmcli/views/view"
	"swarmcli/views/viewstack"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Model holds app state
type Model struct {
	viewport viewport.Model

	systemInfo *systeminfoview.Model

	currentView view.View
	viewStack   viewstack.Stack

	commandInput *commandinput.Model
}

func InitialModel() *Model {
	vp := viewport.New(80, 20)
	vp.YPosition = 5

	loading := loadingview.New(80, 20, true, map[string]string{
		"title":   "Initializing",
		"header":  "Fetching cluster info",
		"message": "Loading Swarm nodes and stacks...",
	})

	return &Model{
		viewport:     vp,
		currentView:  loading,
		systemInfo:   systeminfoview.New(version),
		viewStack:    viewstack.Stack{},
		commandInput: cmdBar(),
	}
}

// Init  will be automatically called by Bubble Tea if the model implements the Model interface
// and is passed into the tea.NewProgram function.
func (m *Model) Init() tea.Cmd {
	// "" loads all stacks on all nodes
	return tea.Batch(tick(), loadSnapshotAsync(), systeminfoview.LoadStatus())
}

func (m *Model) switchToView(name string, data any) tea.Cmd {
	factory, ok := viewRegistry[name]
	if !ok {
		return nil
	}

	// Exit hook for current view
	exitCmd := m.currentView.OnExit()

	newView, loadCmd := factory(m.viewport.Width, m.viewport.Height, data)
	resizeCmd := handleViewResize(newView, m.viewport.Width, m.viewport.Height)

	// Push current view onto stack
	m.viewStack.Push(m.currentView)
	m.currentView = newView

	// Enter hook for new view
	enterCmd := newView.OnEnter()

	return tea.Batch(exitCmd, resizeCmd, loadCmd, enterCmd)
}

func (m *Model) replaceView(name string, data any) tea.Cmd {
	factory, ok := viewRegistry[name]
	if !ok {
		return nil
	}

	// Run exit hook on current view
	exitCmd := m.currentView.OnExit()

	newView, loadCmd := factory(m.viewport.Width, m.viewport.Height, data)
	resizeCmd := handleViewResize(newView, m.viewport.Width, m.viewport.Height)

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
