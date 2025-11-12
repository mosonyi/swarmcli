package node

import (
	"swarmcli/args"
	"swarmcli/registry"
	nodesview "swarmcli/views/nodes"

	tea "github.com/charmbracelet/bubbletea"
)
import "swarmcli/views/view"

type DockerNodeLs struct{}

func (DockerNodeLs) Name() string        { return "node ls" }
func (DockerNodeLs) Description() string { return "docker node ls" }

func (DockerNodeLs) Execute(ctx any, args args.Args) tea.Cmd {
	return func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: nodesview.ViewName,
			Payload:  nil,
		}
	}
}

var lsCmd = DockerNodeLs{}

func init() {
	registry.Register(lsCmd)
}
