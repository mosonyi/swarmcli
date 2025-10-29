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
		//apiCtx := ctx.(api.Context)
		//if len(args.Positionals) == 0 {
		//	return view.ErrorMsg(fmt.Sprintf("Usage: %s <node-id>", c.Name()))
		//}

		nodeID := args.Positionals[0]
		verbose := args.Has("verbose")

		inspectContent, _ := docker.Inspect(context.Background(), docker.InspectNode, nodeID)
		//if err != nil {
		//	return view.ErrorMsg(fmt.Sprintf("Failed to inspect node %q: %v", nodeID, err))
		//}

		if verbose {
			inspectContent = fmt.Sprintf("Verbose output enabled\n\n%s", inspectContent)
		}

		return view.NavigateToMsg{
			ViewName: inspectview.ViewName,
			Payload:  inspectContent,
		}
	}
}

var InspectCmd = DockerNodeInspect{}

func init() {
	registry.Register(InspectCmd)
}
