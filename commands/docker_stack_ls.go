package commands

import tea "github.com/charmbracelet/bubbletea"
import "swarmcli/views/view"

type DockerStackLs struct{}

func (DockerStackLs) Name() string        { return "docker stack ls" }
func (DockerStackLs) Description() string { return "List all Docker stacks" }

func (DockerStackLs) Execute(ctx Context, args []string) tea.Cmd {
	app := ctx.App.(view.Navigator)
	return app.NavigateTo("stacks", nil)
}
