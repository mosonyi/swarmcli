// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package volume

import (
	"github.com/Eldara-Tech/swarmcli/v2/args"
	"github.com/Eldara-Tech/swarmcli/v2/registry"
	"github.com/Eldara-Tech/swarmcli/v2/views/view"
	volumesview "github.com/Eldara-Tech/swarmcli/v2/views/volumes"

	tea "github.com/charmbracelet/bubbletea"
)

type DockerVolumeLs struct{}

func (DockerVolumeLs) Name() string        { return "volume" }
func (DockerVolumeLs) Description() string { return "docker volume ls" }

func (DockerVolumeLs) Spec() registry.CommandSpec {
	return registry.CommandSpec{
		Detail: "Opens the Docker Volumes list, where you can browse and " +
			"inspect volumes. The list covers the connected node only; " +
			"listing volumes across all swarm nodes is a Business Edition " +
			"feature.",
		Examples: []string{":volume"},
	}
}

func (DockerVolumeLs) Execute(ctx any, args args.Args) tea.Cmd {
	return func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: volumesview.ViewName,
			Payload:  nil,
		}
	}
}

var lsCmd = DockerVolumeLs{}

func init() {
	registry.Register(lsCmd)
}
