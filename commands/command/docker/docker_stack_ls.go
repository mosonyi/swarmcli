package docker

import (
	"swarmcli/registry"
	stacksview "swarmcli/views/stacks"

	tea "github.com/charmbracelet/bubbletea"
)
import "swarmcli/views/view"

type DockerStackLs struct{}

func (DockerStackLs) Name() string        { return "docker stack ls" }
func (DockerStackLs) Description() string { return "List all Docker stacks" }

func (DockerStackLs) Execute(ctx any, args []string) tea.Cmd {
	return func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: stacksview.ViewName,
			Payload:  nil,
		}
	}
}

var stackLsCmd = DockerStackLs{}

func init() {
	registry.Register(stackLsCmd)
}
