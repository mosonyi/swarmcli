// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package network

import (
	"swarmcli/args"
	"swarmcli/registry"
	networksview "swarmcli/views/networks"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

type DockerNetworkLs struct{}

func (DockerNetworkLs) Name() string        { return "network" }
func (DockerNetworkLs) Description() string { return "docker network ls" }

func (DockerNetworkLs) Spec() registry.CommandSpec {
	return registry.CommandSpec{
		Detail: "Opens the Docker Networks list, where you can create, " +
			"inspect and delete networks and see which services use a " +
			"given network.",
		Examples: []string{":network"},
	}
}

func (DockerNetworkLs) Execute(ctx any, args args.Args) tea.Cmd {
	return func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: networksview.ViewName,
			Payload:  nil,
		}
	}
}

var lsCmd = DockerNetworkLs{}

func init() {
	registry.Register(lsCmd)
}
