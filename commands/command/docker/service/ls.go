// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package service

import (
	"github.com/Eldara-Tech/swarmcli/v2/args"
	"github.com/Eldara-Tech/swarmcli/v2/registry"
	servicesview "github.com/Eldara-Tech/swarmcli/v2/views/services"
	"github.com/Eldara-Tech/swarmcli/v2/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

type DockerServiceLs struct{}

func (DockerServiceLs) Name() string        { return "service" }
func (DockerServiceLs) Description() string { return "docker service ls" }

func (DockerServiceLs) Spec() registry.CommandSpec {
	return registry.CommandSpec{
		Detail: "Opens the Services list across all stacks (docker service ls), " +
			"where you can browse, scale, restart and inspect every service in " +
			"the swarm.",
		Examples: []string{":service", ":svc"},
	}
}

func (DockerServiceLs) Execute(ctx any, args args.Args) tea.Cmd {
	return func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: servicesview.ViewName,
			Payload:  nil,
		}
	}
}

// svcAlias is the ":svc" alias for ":service". It inherits Description, Execute
// and Spec from the embedded primary; AliasOf folds it under "service" in the
// command list and help.
type svcAlias struct{ DockerServiceLs }

func (svcAlias) Name() string    { return "svc" }
func (svcAlias) AliasOf() string { return "service" }

var lsCmd = DockerServiceLs{}

func init() {
	registry.Register(lsCmd)
	registry.Register(svcAlias{})
}
