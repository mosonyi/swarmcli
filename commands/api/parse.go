package api

import (
	"strings"
	"swarmcli/args"
	"swarmcli/commands"
	"swarmcli/registry"
)

// ParseInput takes a full input string like:
// "docker node inspect node-1 --verbose --limit=10"
// It returns the matching Command and parsed Args.
func ParseInput(input string) (registry.Command, args.Args, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, args.Args{}, ErrEmptyCommand
	}

	parts := strings.Fields(input)

	// Find longest matching command name
	var cmd registry.Command
	var ok bool
	for i := len(parts); i > 0; i-- {
		tryName := strings.Join(parts[:i], " ")
		if c, found := commands.Get(tryName); found {
			cmd = c
			ok = true
			parts = parts[i:] // remaining = args + flags
			break
		}
	}

	if !ok {
		return nil, args.Args{}, ErrUnknownCommand(input)
	}

	parsed := parseArgs(parts)
	return cmd, parsed, nil
}

// parseArgs separates flags (--flag or --flag=value) from positionals.
func parseArgs(parts []string) args.Args {
	args := args.Args{
		Flags:       make(map[string]string),
		Positionals: []string{},
	}

	for _, p := range parts {
		if strings.HasPrefix(p, "--") {
			p = strings.TrimPrefix(p, "--")
			if eq := strings.Index(p, "="); eq != -1 {
				args.Flags[p[:eq]] = p[eq+1:]
			} else {
				args.Flags[p] = "true"
			}
		} else {
			args.Positionals = append(args.Positionals, p)
		}
	}

	return args
}
