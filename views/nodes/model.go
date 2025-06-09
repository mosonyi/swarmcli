package nodesview

import (
	"fmt"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"swarmcli/docker"
)

type Model struct {
	viewport viewport.Model
	Visible  bool

	nodes  []string
	cursor int

	ready bool
}

type StackService struct {
	StackName   string
	ServiceName string
}

// Create a new instance
func New(width, height int) Model {
	vp := viewport.New(width, height)
	vp.SetContent("")
	return Model{
		viewport: vp,
		Visible:  false,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func LoadNodes() tea.Cmd {
	return func() tea.Msg {
		var list []string
		nodes, _ := docker.ListSwarmNodes()
		for _, n := range nodes {
			list = append(list, fmt.Sprint(n))
		}
		return Msg(list)
	}
}
