// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package api //nolint:revive // standard short package name

import (
	"fmt"
	"github.com/Eldara-Tech/swarmcli/v2/args"
	"github.com/Eldara-Tech/swarmcli/v2/commands"
	"github.com/Eldara-Tech/swarmcli/v2/registry"
	"strings"
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

	spec, hasSpec := registry.SpecOf(cmd)

	// Flags declared with TakesValue consume the following token
	// (`--host localhost`), in addition to the `--host=localhost` form.
	valueFlags := map[string]bool{}
	if hasSpec {
		for _, f := range spec.Flags {
			if f.TakesValue {
				valueFlags[f.Name] = true
				if f.Short != "" {
					valueFlags[f.Short] = true
				}
			}
		}
	}

	parsed, err := parseArgs(parts, valueFlags)
	if err != nil {
		return nil, args.Args{}, err
	}

	// Passthrough commands keep their own messaging and must not have
	// their args inspected here (e.g. the OSS bootstrap stub).
	if hasSpec && spec.Passthrough {
		return cmd, parsed, nil
	}

	// Help wins over flag validation: `:cmd --help --bogus` shows help.
	// `--help` is a normal long flag; `-h`/`-help` land in positionals
	// because parseArgs only strips "--", so check there too.
	if parsed.Has("help") || hasShortHelp(parsed.Positionals) {
		return nil, args.Args{}, ErrHelpRequested{Cmd: cmd}
	}

	if hasSpec {
		if err := validateFlags(cmd.Name(), spec, parsed); err != nil {
			return nil, args.Args{}, err
		}
	}

	return cmd, parsed, nil
}

// hasShortHelp reports whether positionals contain a single-dash help
// token. parseArgs leaves single-dash tokens as positionals, so this
// recognises `-h`/`-help` without changing general parser semantics.
func hasShortHelp(positionals []string) bool {
	for _, p := range positionals {
		if p == "-h" || p == "-help" {
			return true
		}
	}
	return false
}

// parseArgs separates flags from positionals. A flag is `--name`,
// `--name=value`, or — when name is in valueFlags — `--name value`
// (the next token is consumed as the value). A value flag at the end
// of input with no following token is an error.
func parseArgs(parts []string, valueFlags map[string]bool) (args.Args, error) {
	out := args.Args{
		Flags:       make(map[string]string),
		Positionals: []string{},
	}

	for i := 0; i < len(parts); i++ {
		p := parts[i]
		if !strings.HasPrefix(p, "--") {
			out.Positionals = append(out.Positionals, p)
			continue
		}

		name := strings.TrimPrefix(p, "--")
		if eq := strings.Index(name, "="); eq != -1 {
			out.Flags[name[:eq]] = name[eq+1:]
			continue
		}
		if valueFlags[name] {
			if i+1 >= len(parts) {
				return args.Args{}, fmt.Errorf("flag --%s requires a value", name)
			}
			i++
			out.Flags[name] = parts[i]
			continue
		}
		out.Flags[name] = "true"
	}

	return out, nil
}
