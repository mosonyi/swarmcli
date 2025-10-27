package commands

import "strings"

var registry = map[string]Command{}

func Register(cmd Command) {
	registry[cmd.Name()] = cmd
}

func Get(name string) (Command, bool) {
	name = strings.TrimSpace(strings.TrimPrefix(name, ":"))
	cmd, ok := registry[name]
	return cmd, ok
}

func List() []Command {
	cmds := make([]Command, 0, len(registry))
	for _, c := range registry {
		cmds = append(cmds, c)
	}
	return cmds
}

func Suggestions(prefix string) []string {
	prefix = strings.TrimSpace(strings.TrimPrefix(prefix, ":"))
	suggestions := []string{}
	for name := range registry {
		if strings.HasPrefix(name, prefix) {
			suggestions = append(suggestions, name)
		}
	}
	return suggestions
}
