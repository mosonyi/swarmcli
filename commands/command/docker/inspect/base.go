package inspect

import (
	"context"
	"fmt"
	"path"
	"runtime"
	"strings"
	"swarmcli/args"
	"swarmcli/docker"
	inspectview "swarmcli/views/inspect"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

type DockerInspectBase struct {
	Type        docker.InspectType
	CommandName string
	Desc        string
}

// inferFromPackage figures out the command name and type from the package path.
func (c *DockerInspectBase) inferFromPackage() {
	if c.CommandName != "" && c.Type != "" {
		return
	}

	pc, _, _, ok := runtime.Caller(2)
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
	idx := strings.Index(pkgPath, "docker/")
	if idx == -1 {
		c.CommandName = "docker inspect"
		c.Type = "unknown"
		return
	}

	segments := strings.Split(pkgPath[idx:], "/")
	c.CommandName = strings.Join(segments, " ")
	if len(segments) >= 2 {
		c.Type = docker.InspectType(segments[1])
	} else {
		c.Type = "unknown"
	}
}

// Name implements registry.Command
func (c DockerInspectBase) Name() string {
	c.inferFromPackage()
	return c.CommandName
}

// Description implements registry.Command
func (c DockerInspectBase) Description() string {
	return c.Desc
}

// Execute implements registry.Command
func (c DockerInspectBase) Execute(ctx any, a args.Args) tea.Cmd {
	return func() tea.Msg {
		c.inferFromPackage()

		//if len(a.Positionals) < 1 {
		//	return view.ErrorMsg{
		//		Message: fmt.Sprintf("Usage: %s <ID>", c.Name()),
		//	}
		//}

		id := a.Positionals[0]

		jsonStr, _ := docker.Inspect(context.Background(), c.Type, id)
		//if err != nil {
		//	return view.ErrorMsg{
		//		Message: fmt.Sprintf("Failed to inspect %s %q: %v", c.Type, id, err),
		//	}
		//}

		return view.NavigateToMsg{
			ViewName: inspectview.ViewName,
			Payload: map[string]interface{}{
				"title": fmt.Sprintf("%s: %s", c.Type, id),
				"json":  jsonStr,
			},
		}
	}
}
