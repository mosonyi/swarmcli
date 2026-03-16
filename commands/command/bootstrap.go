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
