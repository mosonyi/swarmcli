// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package node

import (
	"github.com/Eldara-Tech/swarmcli/args"
	"github.com/Eldara-Tech/swarmcli/registry"
	nodesview "github.com/Eldara-Tech/swarmcli/views/nodes"

	tea "github.com/charmbracelet/bubbletea"
)
import "github.com/Eldara-Tech/swarmcli/views/view"

type DockerNodeLs struct{}

func (DockerNodeLs) Name() string        { return "node" }
func (DockerNodeLs) Description() string { return "docker node ls" }

func (DockerNodeLs) Spec() registry.CommandSpec {
	return registry.CommandSpec{
		Detail: "Opens the cluster Nodes list, where you can inspect " +
			"nodes, change availability, promote or demote managers, " +
			"manage node labels, view a node's tasks, and remove nodes " +
			"from the swarm.",
		Examples: []string{":node"},
	}
}

func (DockerNodeLs) Execute(ctx any, args args.Args) tea.Cmd {
	return func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: nodesview.ViewName,
			Payload:  nil,
		}
	}
}

var lsCmd = DockerNodeLs{}

func init() {
	registry.Register(lsCmd)
}
