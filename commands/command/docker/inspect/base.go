package inspect

import (
	"fmt"
	"path"
	"runtime"
	"strings"
	"swarmcli/commands/api"
	"swarmcli/docker"
	inspectview "swarmcli/views/inspect"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

type DockerInspectBase struct {
	Type        docker.InspectType
	CommandName string
	Description string
}

// inferFromPackage figures out both the command name (docker node inspect)
// and the resource type (node, service, container, etc.) based on the file path.
func (c *DockerInspectBase) inferFromPackage() {
	pc, _, _, ok := runtime.Caller(2) // skip two levels up for stable call site
	if !ok {
		c.CommandName = "docker inspect"
		c.Type = "unknown"
		return
	}
	fn := runtime.FuncForPC(pc)
	if fn == nil {
		c.CommandName = "docker inspect"
		c.Type = "unknown"
		return
	}

	pkgPath := path.Dir(fn.Name())

	// Look for "docker/" in path
	idx := strings.Index(pkgPath, "docker/")
	if idx == -1 {
		c.CommandName = "docker inspect"
		c.Type = "unknown"
		return
	}

	// Extract e.g. "docker/node/inspect"
	cmdPart := pkgPath[idx:]
	segments := strings.Split(cmdPart, "/")

	// Build command name and type
	c.CommandName = strings.Join(segments, " ")

	if len(segments) >= 2 {
		c.Type = docker.InspectType(segments[1]) // "node", "service", etc.
	} else {
		c.Type = "unknown"
	}
}

// Name returns the inferred command name.
func (c DockerInspectBase) Name() string {
	if c.CommandName == "" {
		tmp := c
		tmp.inferFromPackage()
		return tmp.CommandName
	}
	return c.CommandName
}

func (c DockerInspectBase) DescriptionText() string {
	return c.Description
}

func (c DockerInspectBase) Execute(ctx api.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		if c.CommandName == "" || c.Type == "" || c.Type == "unknown" {
			tmp := c
			tmp.inferFromPackage()
			c.CommandName = tmp.CommandName
			c.Type = tmp.Type
		}

		if len(args) < 1 {
			//return view.ErrorMsg{Message: fmt.Sprintf("Usage: %s <ID>", c.Name())}
		}

		id := args[0]

		jsonStr, err := docker.Inspect(ctx, c.Type, id)
		if err != nil {
			//return view.ErrorMsg{Message: fmt.Sprintf("Failed to inspect %s %q: %v", c.Type, id, err)}
		}

		return view.NavigateToMsg{
			ViewName: inspectview.ViewName,
			Payload: map[string]interface{}{
				"title": fmt.Sprintf("%s: %s", c.Type, id),
				"json":  jsonStr,
			},
		}
	}
}
