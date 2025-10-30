package node

import (
	"context"
	"fmt"
	"swarmcli/args"
	"swarmcli/docker"
	"swarmcli/registry"
	"swarmcli/views/inspect"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

type DockerNodeInspect struct{}

func (c DockerNodeInspect) Name() string {
	return "docker node inspect"
}

func (c DockerNodeInspect) Description() string {
	return "Inspect details about a Docker node"
}

func (c DockerNodeInspect) Execute(ctx any, args args.Args) tea.Cmd {
	return func() tea.Msg {
		nodeID := args.Positionals[0]
		verbose := args.Has("verbose")

		inspectContent, err := docker.Inspect(context.Background(), docker.InspectNode, nodeID)
		if err != nil {
			inspectContent = fmt.Sprintf("Error inspecting node %q: %v", nodeID, err)
		}

		if verbose {
			inspectContent = fmt.Sprintf("Verbose output enabled\n\n%s", inspectContent)
		}

		return view.NavigateToMsg{
			ViewName: inspectview.ViewName,
			Payload: map[string]interface{}{
				"title": fmt.Sprintf("Node: %s", nodeID),
				"json":  inspectContent,
			},
		}
	}
}

var InspectCmd = DockerNodeInspect{}

func init() {
	registry.Register(InspectCmd)
}
