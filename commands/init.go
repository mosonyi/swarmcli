package commands

import (
	"swarmcli/commands/command"
	"swarmcli/commands/command/docker"
	"swarmcli/commands/command/docker/node"
	"swarmcli/registry"
)

func Init() {
	registry.Register(command.Help{})
	registry.Register(docker.DockerStackLs{})
	registry.Register(node.DockerNodeLs{})
	registry.Register(node.DockerNodeInspect{})
}

// Public passthroughs so app code can just use `commands.Get()` or `commands.All()`
func Register(cmd registry.Command)            { registry.Register(cmd) }
func Get(name string) (registry.Command, bool) { return registry.Get(name) }
func All() []registry.Command                  { return registry.All() }
func Suggest(prefix string) []string           { return registry.Suggest(prefix) }
