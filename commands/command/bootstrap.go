// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package command

import (
	"swarmcli/args"
	"swarmcli/registry"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

type Bootstrap struct{}

func (Bootstrap) Name() string { return "bootstrap" }
func (Bootstrap) Description() string {
	return view.BEHelpDesc("bootstrap", "Deploy swarmcli infrastructure (rbac-proxy + agent)")
}

// Spec marks the OSS stub as passthrough: help interception and strict
// flag validation are skipped so every invocation (incl. --help and any
// Pro flags) reaches Execute and yields the Business-Edition notice.
// This keeps Pro flag names/descriptions out of the OSS repo.
func (Bootstrap) Spec() registry.CommandSpec {
	return registry.CommandSpec{Passthrough: true}
}

func (Bootstrap) Execute(_ any, _ args.Args) tea.Cmd {
	return func() tea.Msg {
		return view.AppErrorMsg{
			Error: view.BEUnavailableErr("Bootstrap").Error(),
		}
	}
}

func init() {
	registry.Register(Bootstrap{})
}
