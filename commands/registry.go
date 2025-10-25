package commands

var registry = map[string]Command{}

func Register(cmd Command) {
	registry[cmd.Name()] = cmd
}

func Get(name string) (Command, bool) {
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
