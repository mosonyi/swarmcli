package node

import (
	"swarmcli/registry"
	nodesview "swarmcli/views/nodes"

	tea "github.com/charmbracelet/bubbletea"
)
import "swarmcli/views/view"

type DockerNodeLs struct{}

func (DockerNodeLs) Name() string        { return "docker node ls" }
func (DockerNodeLs) Description() string { return "List all Docker nodes" }

func (DockerNodeLs) Execute(ctx any, args []string) tea.Cmd {
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
