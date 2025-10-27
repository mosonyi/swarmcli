package docker

import (
	"swarmcli/commands/api"
	stacksview "swarmcli/views/stacks"

	tea "github.com/charmbracelet/bubbletea"
)
import "swarmcli/views/view"

type DockerStackLs struct{}

func (DockerStackLs) Name() string        { return "docker stack ls" }
func (DockerStackLs) Description() string { return "List all Docker stacks" }

func (DockerStackLs) Execute(ctx api.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		// Todo: implement stacks view to get stacks from all nodes
		return view.NavigateToMsg{
			ViewName: stacksview.ViewName,
			Payload:  nil,
		}
	}
}
