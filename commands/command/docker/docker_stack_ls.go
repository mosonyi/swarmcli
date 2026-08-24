// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"github.com/Eldara-Tech/swarmcli/v2/args"
	"github.com/Eldara-Tech/swarmcli/v2/registry"
	stacksview "github.com/Eldara-Tech/swarmcli/v2/views/stacks"

	tea "github.com/charmbracelet/bubbletea"
)
import "github.com/Eldara-Tech/swarmcli/v2/views/view"

type DockerStackLs struct{}

func (DockerStackLs) Name() string        { return "stack" }
func (DockerStackLs) Description() string { return "List all Docker stacks: docker stack ls" }

func (DockerStackLs) Spec() registry.CommandSpec {
	return registry.CommandSpec{
		Detail: "Opens the Stacks list (docker stack ls), where you can " +
			"create, edit, inspect and delete stacks, view a stack's " +
			"tasks, and drill into a stack to manage its services.",
		Examples: []string{":stack"},
	}
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
