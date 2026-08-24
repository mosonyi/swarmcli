// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package command

import (
	"github.com/Eldara-Tech/swarmcli/v2/args"
	"github.com/Eldara-Tech/swarmcli/v2/registry"

	tea "github.com/charmbracelet/bubbletea"
)

// aliasCommand is a simple wrapper to provide aliases for commands
type aliasCommand struct {
	name   string
	target registry.Command
}

func (a aliasCommand) Name() string        { return a.name }
func (a aliasCommand) Description() string { return a.target.Description() }
func (a aliasCommand) Execute(ctx any, args args.Args) tea.Cmd {
	return a.target.Execute(ctx, args)
}

// AliasOf implements registry.Aliaser.
func (a aliasCommand) AliasOf() string { return a.target.Name() }
