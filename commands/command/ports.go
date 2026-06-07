// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package command

import (
	"swarmcli/args"
	"swarmcli/registry"
	portsview "swarmcli/views/ports"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

type Ports struct{}

func (Ports) Name() string        { return "ports" }
func (Ports) Description() string { return "Show required Swarm cluster ports" }

func (Ports) Spec() registry.CommandSpec {
	return registry.CommandSpec{
		Detail: "Opens the required Swarm cluster ports (node-to-node) reference view, " +
			"showing which ports must be open between Swarm nodes (managers and workers) for " +
			"cluster management, node communication, and overlay network VXLAN traffic.",
		Examples: []string{":ports", ":port"},
	}
}

func (Ports) Execute(ctx any, args args.Args) tea.Cmd {
	return func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: portsview.ViewName,
			Payload:  nil,
		}
	}
}

var portsCmd = Ports{}

func init() {
	registry.Register(portsCmd)
	registry.Register(aliasCommand{name: "port", target: portsCmd})
}
