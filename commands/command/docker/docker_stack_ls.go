// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"swarmcli/args"
	"swarmcli/registry"
	stacksview "swarmcli/views/stacks"

	tea "github.com/charmbracelet/bubbletea"
)
import "swarmcli/views/view"

type DockerStackLs struct{}

func (DockerStackLs) Name() string        { return "stack" }
func (DockerStackLs) Description() string { return "List all Docker stacks: docker stack ls" }

func (DockerStackLs) Spec() registry.CommandSpec {
	return registry.CommandSpec{Examples: []string{":stack"}}
}

func (DockerStackLs) Execute(ctx any, args args.Args) tea.Cmd {
	return func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: stacksview.ViewName,
			Payload:  nil,
		}
	}
}

var stackLsCmd = DockerStackLs{}

func init() {
	registry.Register(stackLsCmd)
}
