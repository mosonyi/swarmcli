// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package command

import (
	"swarmcli/args"
	"swarmcli/registry"
	contextsview "swarmcli/views/contexts"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

type Contexts struct{}

func (Contexts) Name() string        { return "contexts" }
func (Contexts) Description() string { return "List and switch Docker contexts" }

func (Contexts) Spec() registry.CommandSpec {
	return registry.CommandSpec{
		Detail: "Opens the Docker context list, where you can switch the " +
			"active context (which reloads the cluster) and create, " +
			"inspect, edit, delete, import or export contexts.",
		Examples: []string{":contexts"},
	}
}

func (Contexts) Execute(ctx any, args args.Args) tea.Cmd {
	return func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: contextsview.ViewName,
			Payload:  nil,
		}
	}
}

var contextsCmd = Contexts{}

func init() {
	registry.Register(contextsCmd)
	// Register aliases
	registry.Register(aliasCommand{name: "context", target: contextsCmd})
	registry.Register(aliasCommand{name: "ctx", target: contextsCmd})
}
