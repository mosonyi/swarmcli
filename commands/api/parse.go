// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package api //nolint:revive // standard short package name

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

	spec, hasSpec := registry.SpecOf(cmd)

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
