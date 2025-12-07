package command

import (
	"swarmcli/args"
	"swarmcli/registry"

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
