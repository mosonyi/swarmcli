package commands

import (
	"strings"
)

var registry = make(map[string]Command)

// Register a command globally, usually called from init()
func Register(cmd Command) {
	registry[cmd.Name()] = cmd
}

// Get returns a command by exact name
func Get(name string) (Command, bool) {
	c, ok := registry[name]
	return c, ok
}

// List returns all registered commands
func List() []Command {
	cmds := make([]Command, 0, len(registry))
	for _, c := range registry {
		cmds = append(cmds, c)
	}
	return cmds
}

// Suggest returns all command names that start with a given prefix
func Suggest(prefix string) []string {
	var out []string
	for name := range registry {
		if prefix == "" || strings.HasPrefix(name, prefix) {
			out = append(out, name)
		}
	}
	return out
}
