package docker

import (
	"swarmcli/commands/api"

	tea "github.com/charmbracelet/bubbletea"
)
import "swarmcli/views/view"

type DockerStackLs struct{}

func (DockerStackLs) Name() string        { return "docker stack ls" }
func (DockerStackLs) Description() string { return "List all Docker stacks" }

func (DockerStackLs) Execute(ctx api.Context, args []string) tea.Cmd {
	app := ctx.App.(view.Navigator)
	return app.NavigateTo("stacks", nil)
}
