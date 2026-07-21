// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package registry

import (
	"github.com/Eldara-Tech/swarmcli/args"
	"sort"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
)

type Command interface {
	Name() string
	Description() string
	Execute(ctx any, args args.Args) tea.Cmd
}

// apiRegistry stores all registered commands. It is only written to during
// init() (single-threaded) and read-only after program start, so no mutex
// is needed.
var apiRegistry = map[string]Command{}

// Register a new command. Must only be called from init() functions.
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

// Aliaser is optionally implemented by alias commands to indicate
// which primary command they delegate to.
type Aliaser interface {
	AliasOf() string
}

// CommandWithAliases pairs a primary command with its collected alias names.
type CommandWithAliases struct {
	Command
	Aliases []string
}

// PrimaryCommands returns deduplicated commands with alias names collected.
// Commands implementing Aliaser are folded into their target's Aliases slice.
func PrimaryCommands() []CommandWithAliases {
	aliases := map[string][]string{} // target name → alias names
	primaries := map[string]Command{}

	for _, cmd := range apiRegistry {
		if a, ok := cmd.(Aliaser); ok {
			aliases[a.AliasOf()] = append(aliases[a.AliasOf()], cmd.Name())
		} else {
			primaries[cmd.Name()] = cmd
		}
	}

	out := make([]CommandWithAliases, 0, len(primaries))
	for name, cmd := range primaries {
		a := aliases[name]
		sort.Strings(a)
		out = append(out, CommandWithAliases{
			Command: cmd,
			Aliases: a,
		})
	}
	return out
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
