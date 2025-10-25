package commands

import (
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

type DockerStackLsCommand struct{}

func (DockerStackLsCommand) Name() string        { return "stack ls" }
func (DockerStackLsCommand) Description() string { return "List all stacks on all nodes" }

func (DockerStackLsCommand) Execute(ctx Context, args []string) tea.Cmd {
	return func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: "stacks",
			Payload:  nil,
		}
	}
}

func init() {
	Register(DockerStackLsCommand{})
}
