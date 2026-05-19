// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package command

import (
	"swarmcli/args"
	"swarmcli/registry"

	tea "github.com/charmbracelet/bubbletea"
)

type Quit struct{}

func (Quit) Name() string        { return "quit" }
func (Quit) Description() string { return "Exit SwarmCLI" }

func (Quit) Spec() registry.CommandSpec {
	return registry.CommandSpec{
		Detail:   "Exits SwarmCLI immediately, without a confirmation prompt.",
		Examples: []string{":quit"},
	}
}

func (Quit) Execute(_ any, _ args.Args) tea.Cmd {
	return tea.Quit
}

var quitCmd = Quit{}

func init() {
	registry.Register(quitCmd)
	registry.Register(aliasCommand{name: "q", target: quitCmd})
}
