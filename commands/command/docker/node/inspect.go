package node

import (
	"swarmcli/commands/command/docker/inspect"
	"swarmcli/registry"
)

type DockerNodeInspect struct {
	inspect.DockerInspectBase
}

// NewDockerNodeInspect creates a ready-to-register node inspect command.
func NewDockerNodeInspect() DockerNodeInspect {
	return DockerNodeInspect{
		DockerInspectBase: inspect.DockerInspectBase{
			Desc: "Inspect a Docker node by ID",
		},
	}
}

// Optional: override Name if you want a custom command name
// func (c DockerNodeInspect) Name() string { return "docker node inspect" }

// Optional: override Execute if you need extra behavior
// func (c DockerNodeInspect) Execute(ctx api.Context, a args.Args) tea.Cmd {
//     return c.DockerInspectBase.Execute(ctx, a)
// }

var nodeInspect = NewDockerNodeInspect()

func init() {
	registry.Register(nodeInspect)
}
