// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package registry

// FlagSpec declares a single flag a command accepts. It drives both the
// per-command help screen and strict unknown-flag validation.
type FlagSpec struct {
	Name        string // long flag, without leading "--" (e.g. "node-id")
	Short       string // optional single-char alias without "-"; "" = none
	TakesValue  bool   // false ⇒ boolean (--force); true ⇒ --host <value>
	Placeholder string // usage value placeholder, e.g. "<host>"; used only if TakesValue
	Description string
}

// CommandSpec is the optional, declarative documentation/validation
// contract for a command. Commands expose it via CommandWithSpec.
type CommandSpec struct {
	// Usage is the positional usage, e.g. "[command]" or "" for none.
	Usage string
	// Detail is optional prose rendered under USAGE — explain modes,
	// e.g. that running with no flags starts an interactive flow.
	Detail string
	// Flags is the accepted-flag allow-list for strict validation.
	Flags []FlagSpec
	// Examples are full example invocations, incl. the leading ':'.
	Examples []string

	// Passthrough disables both help interception and strict flag
	// validation for this command: every argument reaches Execute
	// unchanged. Intended only for delegating/unavailable stubs (e.g.
	// the OSS bootstrap stub) that must keep their own messaging and
	// must not document Pro internals in the OSS repo.
	Passthrough bool
}

// CommandWithSpec is the optional capability interface. It mirrors the
// Aliaser pattern: discovered via type assertion, additive and
// non-breaking across the swarmcli ↔ swarmcli-be module boundary.
type CommandWithSpec interface {
	Spec() CommandSpec
}

// SpecOf resolves a command's spec, following Aliaser → primary so an
// alias transparently inherits its target's documentation and
// validation rules.
func SpecOf(cmd Command) (CommandSpec, bool) {
	if s, ok := cmd.(CommandWithSpec); ok {
		return s.Spec(), true
	}
	if a, ok := cmd.(Aliaser); ok {
		if primary, found := Get(a.AliasOf()); found {
			return SpecOf(primary)
		}
	}
	return CommandSpec{}, false
}

// Distance is the Levenshtein edit distance between a and b. It backs
// the "did you mean --x?" suggestions for unknown flags and unknown
// :help <command> names. Kept here so both the api (validation) and
// command (help) packages can use it without an import cycle.
func Distance(a, b string) int {
	if a == b {
		return 0
	}
	if len(a) == 0 {
		return len(b)
	}
	if len(b) == 0 {
		return len(a)
	}

	prev := make([]int, len(b)+1)
	curr := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		curr[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			curr[j] = min3(curr[j-1]+1, prev[j]+1, prev[j-1]+cost)
		}
		prev, curr = curr, prev
	}
	return prev[len(b)]
}

func min3(a, b, c int) int {
	m := a
	if b < m {
		m = b
	}
	if c < m {
		m = c
	}
	return m
}
