package commands

import (
	"swarmcli/commands/api"
	"swarmcli/commands/command"
	"swarmcli/commands/command/docker"
	"swarmcli/commands/command/docker/node"
	"swarmcli/commands/internal/registry"
)

func Init() {
	registry.Register(command.Help{})
	registry.Register(docker.DockerStackLs{})
	registry.Register(node.DockerNodeLs{})
}

// Public passthroughs so app code can just use `commands.Get()` or `commands.All()`
func Register(cmd api.Command)            { registry.Register(cmd) }
func Get(name string) (api.Command, bool) { return registry.Get(name) }
func All() []api.Command                  { return registry.All() }
func Suggest(prefix string) []string      { return registry.Suggest(prefix) }
