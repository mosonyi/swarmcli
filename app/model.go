package app

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
type Model struct {
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
func InitialModel() Model {
	vp := viewport.New(80, 20)
	vp.YPosition = 5

	nodes := nodesview.New(80, 20)

	return Model{
		mode:        modeNodes,
		viewport:    vp,
		currentView: nodes,
		systemInfo:  systeminfoview.New(version),
		viewStack:   viewstack.Stack{},
	}
}

// Init  will be automatically called by Bubble Tea if the model implements the Model interface
// and is passed into the tea.NewProgram function.
func (m Model) Init() tea.Cmd {
	return tea.Batch(tick(), nodesview.LoadNodes(), systeminfoview.LoadStatus())
}

func (m Model) switchToView(name string, data any) (Model, tea.Cmd) {
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

func (m Model) renderStackBar() string {
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
