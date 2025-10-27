package commands

type Registry struct {
	cmds map[string]Command
}

var globalRegistry = &Registry{cmds: make(map[string]Command)}

// Register adds a command to the global registry
func Register(cmd Command) {
	globalRegistry.cmds[cmd.Name()] = cmd
}

// Get looks up a command by name
func Get(name string) (Command, bool) {
	cmd, ok := globalRegistry.cmds[name]
	return cmd, ok
}

// List returns all registered commands (useful for autocomplete/help)
func List() []Command {
	cmds := make([]Command, 0, len(globalRegistry.cmds))
	for _, c := range globalRegistry.cmds {
		cmds = append(cmds, c)
	}
	return cmds
}
