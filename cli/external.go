// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

import (
	"fmt"
	"sort"
)

// Verbs this binary does not implement.
//
// swarmcli is published as two artefacts built from the same command name:
// this repository's build, and a build from a private extension wrapper that
// links additional packages. The TUI already accepts commands from that
// wrapper — registry.Register is how :bootstrap and :license reach the command
// palette — and until now the non-interactive front-end did not, so a verb an
// operator could type in the TUI had no headless equivalent and could not
// acquire one without editing this switch.
//
// It is a *registration* seam rather than a list of names on purpose. A list
// would put the wrapper's vocabulary in this repository, which is exactly the
// coupling the two-artefact model exists to avoid: this package must not know
// that a licence, or anything else, is a thing. What it knows is that a
// registered verb has a name, one line of summary, and an exit code.
//
// The OSS build registers nothing, so it advertises nothing — `swarmcli help`
// lists what this binary can actually do, which is the property a hollow
// built-in stub would have given up.

// externalCommand is a verb supplied by a build that embeds this module.
type externalCommand struct {
	name    string
	summary string
	run     func(args []string) int
}

// externalCommands is written only from init() (single-threaded, like
// registry.Register) and read-only afterwards, so it needs no mutex.
var externalCommands = map[string]externalCommand{}

// RegisterCommand adds a top-level verb served by a build embedding this
// module. Call it from init(), before Dispatch can run.
//
// name is what the operator types (`swarmcli <name> …`), summary is the one
// line `swarmcli help` shows beside it, and run receives everything after the
// verb and returns the process exit code — 2 for a usage error, 1 for a
// failure, 0 for success, which is the contract every command here already
// keeps (see usageErr and fail).
//
// It panics on a name that is empty, already registered, or one of this
// package's own verbs. All three are wiring errors fixed at compile-authoring
// time and reachable from no input, and the alternative — quietly ignoring the
// registration, or quietly shadowing `version` — is a binary that behaves
// differently from the source somebody is reading. Precedent: sql.Register and
// http.ServeMux both panic on a duplicate.
func RegisterCommand(name, summary string, run func(args []string) int) {
	switch {
	case name == "":
		panic("cli: RegisterCommand with an empty name")
	case run == nil:
		panic("cli: RegisterCommand(" + name + ") with no run function")
	case builtinCommand(name):
		panic("cli: RegisterCommand(" + name + ") would shadow a built-in command")
	}
	if _, dup := externalCommands[name]; dup {
		panic("cli: RegisterCommand(" + name + ") registered twice")
	}
	externalCommands[name] = externalCommand{name: name, summary: summary, run: run}
}

// builtinCommand reports whether name is served by Dispatch itself. Kept
// beside the switch it mirrors: a verb added there and not here becomes
// shadowable, which is the failure this exists to prevent.
func builtinCommand(name string) bool {
	switch name {
	case "charts", "version", "--version", "-v", "help", "--help", "-h":
		return true
	}
	return false
}

// externalCommandNames returns the registered names in a stable order, so help
// output does not depend on map iteration.
func externalCommandNames() []string {
	names := make([]string, 0, len(externalCommands))
	for n := range externalCommands {
		names = append(names, n)
	}
	sort.Strings(names)
	return names
}

// externalUsage renders the registered verbs as help lines, or "" when none
// are registered — which is every build of this repository on its own.
func externalUsage() string {
	names := externalCommandNames()
	if len(names) == 0 {
		return ""
	}
	out := ""
	for _, n := range names {
		out += fmt.Sprintf("  %-11s %s\n", n, externalCommands[n].summary)
	}
	return out
}
