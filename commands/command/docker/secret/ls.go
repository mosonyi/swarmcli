// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package secret

import (
	"swarmcli/args"
	"swarmcli/registry"
	secretsview "swarmcli/views/secrets"

	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

type DockerSecretLs struct{}

func (DockerSecretLs) Name() string        { return "secret" }
func (DockerSecretLs) Description() string { return "docker secret ls" }

func (DockerSecretLs) Execute(ctx any, args args.Args) tea.Cmd {
	return func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: secretsview.ViewName,
			Payload:  nil,
		}
	}
}

var lsCmd = DockerSecretLs{}

func init() {
	registry.Register(lsCmd)
}
