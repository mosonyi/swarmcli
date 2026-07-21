// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package config

import (
	"github.com/Eldara-Tech/swarmcli/args"
	"github.com/Eldara-Tech/swarmcli/registry"
	configsview "github.com/Eldara-Tech/swarmcli/views/configs"

	tea "github.com/charmbracelet/bubbletea"
)
import "github.com/Eldara-Tech/swarmcli/views/view"

type DockerConfigLs struct{}

func (DockerConfigLs) Name() string        { return "config" }
func (DockerConfigLs) Description() string { return "docker config ls" }

func (DockerConfigLs) Spec() registry.CommandSpec {
	return registry.CommandSpec{
		Detail: "Opens the Docker Configs list, where you can create, " +
			"clone, inspect and delete configs and see which stacks use " +
			"them.",
		Examples: []string{":config"},
	}
}

func (DockerConfigLs) Execute(ctx any, args args.Args) tea.Cmd {
	return func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: configsview.ViewName,
			Payload:  nil,
		}
	}
}

var lsCmd = DockerConfigLs{}

func init() {
	registry.Register(lsCmd)
}
