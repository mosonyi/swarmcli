// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// Per-command help renders from flagDocs, so a flag with no entry there prints
// as a bare name with no explanation — and the option reference silently stops
// covering it.
func TestFlagDocsCoverEveryListedFlag(t *testing.T) {
	documented := map[string]bool{}
	for _, d := range flagDocs {
		require.NotEmpty(t, d.Desc, "%s has no description", d.Name)
		require.False(t, documented[d.Name], "%s is documented twice", d.Name)
		documented[d.Name] = true
	}

	listed := map[string]bool{}
	for _, c := range chartsCommands {
		for _, f := range c.Flags {
			listed[f] = true
			require.True(t, documented[f], "%s takes %s, which the option reference does not describe", c.Name, f)
		}
	}
	for _, d := range flagDocs {
		require.True(t, listed[d.Name], "%s is documented but no command takes it", d.Name)
	}
}

// `charts install --help` printed "Error: unknown flag '--help'" before the
// table existed: only the top-level help was intercepted.
func TestCommandHelpIsInterceptedPerCommand(t *testing.T) {
	var code int
	o, e := capture(t, func() { code = chartsMain([]string{"install", "--help"}) })
	require.Equal(t, 0, code)
	require.Empty(t, e)
	require.Contains(t, o, "swarmcli charts install <release> <chart>")
	require.Contains(t, o, "--set-file")
	// install does not take these, so its help must not offer them.
	require.NotContains(t, o, "--purge-volumes")
	require.NotContains(t, o, "--revision")
}

// Help must survive being asked for after the arguments, which is how anyone
// who started typing the command asks for it.
func TestCommandHelpAfterPositionals(t *testing.T) {
	var code int
	o, _ := capture(t, func() { code = chartsMain([]string{"upgrade", "rel", "repo/chart", "-h"}) })
	require.Equal(t, 0, code)
	require.Contains(t, o, "swarmcli charts upgrade")
}

// A command with no options says so, rather than printing an empty heading.
func TestCommandHelpWithNoOptions(t *testing.T) {
	o, _ := capture(t, func() { chartsMain([]string{"list", "--help"}) })
	require.Contains(t, o, "This command takes no options.")
	require.Contains(t, o, "Alias: ls")
	require.NotContains(t, o, "Options:")
}

// `charts help <command>` and `charts <command> --help` are one page, as
// `:help <cmd>` and `:cmd --help` are in the TUI.
func TestHelpSubcommandMatchesFlag(t *testing.T) {
	viaVerb, _ := capture(t, func() { chartsMain([]string{"help", "status"}) })
	viaFlag, _ := capture(t, func() { chartsMain([]string{"status", "--help"}) })
	require.Equal(t, viaVerb, viaFlag)
	require.Contains(t, viaVerb, "swarmcli charts status <release>")
}

// An unknown name after help falls back to the full usage rather than claiming
// a command exists.
func TestHelpSubcommandUnknownFallsBack(t *testing.T) {
	var code int
	o, _ := capture(t, func() { code = chartsMain([]string{"help", "nonsense"}) })
	require.Equal(t, 0, code)
	require.Contains(t, o, "Usage: swarmcli charts <command> [options]")
}

// "--help" as the VALUE of another flag is a value, not a request for help.
// Only the parser knows the difference, which is why the interception lives
// there rather than in a scan of the arguments.
func TestHelpIsNotConfusedWithAFlagValue(t *testing.T) {
	pos, f, code := parse(cmd(t, "install"), []string{"rel", "repo/chart", "--set", "--help"})
	require.Equal(t, -1, code)
	require.Equal(t, []string{"rel", "repo/chart"}, pos)
	require.Equal(t, []string{"--help"}, f.sets)
}

// The top-level usage keeps the full reference; a command's help shows only its
// own flags. Both come from flagDocs.
func TestOptionReferenceSubsets(t *testing.T) {
	all := renderOptions(nil)
	for _, d := range flagDocs {
		require.Contains(t, all, d.Name)
	}

	only := renderOptions([]string{"--wait", "--timeout"})
	require.Contains(t, only, "--wait")
	require.Contains(t, only, "--timeout")
	require.NotContains(t, only, "--set")

	// Every rendered line wraps rather than running off the terminal.
	for _, l := range strings.Split(strings.TrimRight(all, "\n"), "\n") {
		require.LessOrEqual(t, len([]rune(l)), 80, "option line is too wide: %q", l)
	}
}
