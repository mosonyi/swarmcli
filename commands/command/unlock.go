// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package command

import (
	"github.com/Eldara-Tech/swarmcli/args"
	"github.com/Eldara-Tech/swarmcli/registry"
	"github.com/Eldara-Tech/swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

type Unlock struct{}

func (Unlock) Name() string        { return "unlock" }
func (Unlock) Description() string { return "Unlock a locked Docker Swarm" }

func (Unlock) Spec() registry.CommandSpec {
	return registry.CommandSpec{
		Detail: "Opens a prompt to enter the swarm unlock key and decrypt a " +
			"locked cluster, then reloads its resources. Equivalent to " +
			"`docker swarm unlock`.",
		Examples: []string{":unlock"},
	}
}

func (Unlock) Execute(ctx any, args args.Args) tea.Cmd {
	return func() tea.Msg {
		return view.OpenUnlockDialogMsg{}
	}
}

var unlockCmd = Unlock{}

func init() {
	registry.Register(unlockCmd)
}
