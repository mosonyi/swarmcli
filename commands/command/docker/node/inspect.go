package node

import (
	"fmt"
	"swarmcli/registry"

	tea "github.com/charmbracelet/bubbletea"
)

type DockerNodeInspect struct{}

func (c DockerNodeInspect) Name() string {
	return "docker node inspect"
}

func (c DockerNodeInspect) Description() string {
	return "Inspect details about a Docker node"
}

func (c DockerNodeInspect) Execute(ctx any, args []string) tea.Cmd {
	return func() tea.Msg {
		// You can type assert ctx if you need it:
		// apiCtx := ctx.(api.Context)
		if len(args) == 0 {
			return fmt.Sprintf("Usage: %s <node-id>", c.Name())
		}

		nodeID := args[0]
		// You’ll soon replace this with a call into your docker package:
		return fmt.Sprintf("Inspecting node %s...", nodeID)
	}
}

var InspectCmd = DockerNodeInspect{}

func init() {
	registry.Register(InspectCmd)
}
