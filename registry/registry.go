package registry

import (
	"strings"
	"swarmcli/args"

	tea "github.com/charmbracelet/bubbletea"
)

type Command interface {
	Name() string
	Description() string
	Execute(ctx any, args args.Args) tea.Cmd
}

var apiRegistry = map[string]Command{}

// Register a new command (called from api/init.go)
func Register(cmd Command) {
	apiRegistry[cmd.Name()] = cmd
}

// Get returns a command by name
func Get(name string) (Command, bool) {
	cmd, ok := apiRegistry[name]
	return cmd, ok
}

// All returns a slice of all registered api
func All() []Command {
	cmds := make([]Command, 0, len(apiRegistry))
	for _, c := range apiRegistry {
		cmds = append(cmds, c)
	}
	return cmds
}

// Suggest returns all command names that start with a given prefix
func Suggest(prefix string) []string {
	var out []string
	for name := range apiRegistry {
		if prefix == "" || strings.HasPrefix(name, prefix) {
			out = append(out, name)
		}
	}
	return out
}
