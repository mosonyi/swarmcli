package registry

import (
	"strings"
	"swarmcli/commands/api"
)

var apiRegistry = map[string]api.Command{}

// Register a new command (called from api/init.go)
func Register(cmd api.Command) {
	apiRegistry[cmd.Name()] = cmd
}

// Get returns a command by name
func Get(name string) (api.Command, bool) {
	cmd, ok := apiRegistry[name]
	return cmd, ok
}

// All returns a slice of all registered api
func All() []api.Command {
	cmds := make([]api.Command, 0, len(apiRegistry))
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
