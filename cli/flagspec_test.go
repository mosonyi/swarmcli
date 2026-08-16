// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

import (
	"os"
	"regexp"
	"sort"
	"testing"

	"github.com/stretchr/testify/require"
)

// sampleValue is what a flag needing one is given, so a rejection test exercises
// the rejection rather than "flag requires a value".
var sampleValue = map[string]string{
	"--values":        "values.yaml",
	"--set":           "a=1",
	"--set-file":      "a=/dev/null",
	"--version":       "1.0.0",
	"--timeout":       "5m",
	"--history-max":   "3",
	"--revision":      "2",
	"--resolve-image": "always",
	"--for-version":   "1.13.0",
}

// argsFor renders a flag as an operator would pass it.
func argsFor(flag string) []string {
	if v, ok := sampleValue[flag]; ok {
		return []string{flag, v}
	}
	return []string{flag}
}

// parserFlags reads the long flags parseArgs accepts straight out of its
// source. Hard-coding them here would make this file agree with itself rather
// than with the parser, and the gap it exists to catch — a flag the parser
// learns and no command lists — would be invisible.
func parserFlags(t *testing.T) []string {
	t.Helper()
	src, err := os.ReadFile("args.go")
	require.NoError(t, err)

	seen := map[string]bool{}
	for _, m := range regexp.MustCompile(`case "(--[a-z-]+)"`).FindAllStringSubmatch(string(src), -1) {
		seen[m[1]] = true
	}
	// The switch pairs "-f" with "--values" on one case line, which the pattern
	// above catches by its long half.
	var out []string
	for f := range seen {
		out = append(out, f)
	}
	sort.Strings(out)
	require.NotEmpty(t, out)
	return out
}

// A flag the parser accepts but no command lists is unreachable: every
// invocation of it is rejected, whichever command an operator tries. That is a
// flag shipped by accident, and this is where it surfaces.
func TestEveryParserFlagIsReachable(t *testing.T) {
	listed := map[string]bool{}
	for _, c := range chartsCommands {
		for _, f := range c.Flags {
			listed[f] = true
		}
	}
	for _, f := range parserFlags(t) {
		require.True(t, listed[f], "%s is parsed but no command accepts it", f)
	}
}

// Every flag a row lists must survive that row's allow-list. A typo in the
// table would otherwise reject a flag the handler goes on to read.
func TestAllowedFlagsAreAccepted(t *testing.T) {
	for _, c := range chartsCommands {
		for _, f := range c.Flags {
			t.Run(c.Name+f, func(t *testing.T) {
				_, _, code := parse(c, argsFor(f))
				require.Equal(t, -1, code, "%s should accept %s", c.Name, f)
			})
		}
	}
}

// The defect #451 reports: the flag struct is global, so every subcommand
// parses every flag and silently drops what it does not read — which reads to
// an operator as "it worked".
func TestUnlistedFlagsAreRejected(t *testing.T) {
	all := parserFlags(t)
	for _, c := range chartsCommands {
		allowed := map[string]bool{}
		for _, f := range c.Flags {
			allowed[f] = true
		}
		for _, f := range all {
			if allowed[f] {
				continue
			}
			t.Run(c.Name+f, func(t *testing.T) {
				var code int
				_, e := capture(t, func() { _, _, code = parse(c, argsFor(f)) })
				require.Equal(t, 2, code, "%s should reject %s", c.Name, f)
				require.Contains(t, e, "charts "+c.Name+" does not accept '"+f+"'")
			})
		}
	}
}

// -f is the one short form, and the table lists only long names — so it has to
// be canonicalised before the allow-list sees it, or `-f` is rejected by every
// command that accepts `--values`.
func TestShortFormIsCanonicalised(t *testing.T) {
	c := cmd(t, "install")
	_, _, code := parse(c, []string{"-f", "values.yaml"})
	require.Equal(t, -1, code)

	var rejected int
	_, e := capture(t, func() { _, _, rejected = parse(cmd(t, "list"), []string{"-f", "values.yaml"}) })
	require.Equal(t, 2, rejected)
	require.Contains(t, e, "does not accept '--values'")
}

// A rejection that names only what is wrong leaves an operator guessing. apply
// is the one command with somewhere else to point them.
func TestApplyRejectionNamesTheReleaseFile(t *testing.T) {
	var code int
	_, e := capture(t, func() { _, _, code = parse(cmd(t, "apply"), []string{"--set", "a=1"}) })
	require.Equal(t, 2, code)
	require.Contains(t, e, "only source of truth")
}

// End to end through dispatch, on the two invocations #451 opens with. Neither
// reaches Docker: the allow-list is checked before the handler does anything.
func TestDispatchRejectsIrrelevantFlags(t *testing.T) {
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"list", "--purge-volumes"}, "charts list does not accept '--purge-volumes'"},
		{[]string{"status", "x", "--history-max", "5"}, "charts status does not accept '--history-max'"},
	} {
		var code int
		_, e := capture(t, func() { code = chartsMain(tc.args) })
		require.Equal(t, 2, code)
		require.Contains(t, e, tc.want)
	}
}
