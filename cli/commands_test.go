// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// The table is the source of truth for dispatch, the usage text and the
// generated README blocks, so a malformed row degrades all three at once.
func TestChartsCommands_RowsAreWellFormed(t *testing.T) {
	groups := make(map[string]bool, len(chartGroups))
	for _, g := range chartGroups {
		groups[g] = true
	}

	seen := map[string]string{}
	for _, c := range chartsCommands {
		require.NotEmpty(t, c.Name, "every row needs a name")
		require.NotNil(t, c.Run, "%s has no handler", c.Name)
		require.True(t, groups[c.Group], "%s is in unknown group %q", c.Name, c.Group)
		require.NotEmpty(t, c.Usage, "%s renders no usage line", c.Name)

		for _, n := range append([]string{c.Name}, c.Aliases...) {
			require.Empty(t, seen[n], "%q is claimed by both %s and %s", n, seen[n], c.Name)
			seen[n] = c.Name
		}
		for _, u := range c.Usage {
			require.NotEmpty(t, u.Summary, "%s has a usage line with no summary", c.Name)
			require.NotContains(t, u.Summary, "\n", "%s: a summary is one line", c.Name)
		}
	}
}

// Every command reachable by dispatch must be listed, and everything listed
// must dispatch. This is the invariant the old switch/usage-string pair had no
// way to state.
func TestChartsCommands_LookupCoversEveryNameAndAlias(t *testing.T) {
	for _, n := range commandNames() {
		c, ok := lookupCommand(n)
		require.True(t, ok, "%q is listed but does not dispatch", n)
		require.NotNil(t, c.Run)
	}
	_, ok := lookupCommand("nope")
	require.False(t, ok)
}

func TestChartsUsage_ListsEveryCommand(t *testing.T) {
	usage := chartsUsage()
	for _, c := range chartsCommands {
		require.Contains(t, usage, "  "+c.Name+" ", "%s is missing from the usage text", c.Name)
	}
	// The groups keep their order, so the list stays the walkthrough it reads as.
	last := -1
	for _, g := range chartGroups {
		i := strings.Index(usage, "\n"+g+":\n")
		require.Greater(t, i, last, "group %s is out of order", g)
		last = i
	}
}

// An alias that dispatches but is documented nowhere is how `ls` went four
// releases without a mention.
func TestChartsUsage_ReportsAliases(t *testing.T) {
	require.Contains(t, chartsUsage(), "(alias: ls)")
}

func TestChartsMain_UnknownCommandSuggests(t *testing.T) {
	var code int
	_, e := capture(t, func() { code = chartsMain([]string{"instal"}) })
	require.Equal(t, 2, code)
	require.Contains(t, e, "unknown charts command 'instal'")
	require.Contains(t, e, "did you mean 'install'?")
}

func TestChartsMain_UnknownCommandWithNothingCloseJustReports(t *testing.T) {
	var code int
	_, e := capture(t, func() { code = chartsMain([]string{"xyzzy"}) })
	require.Equal(t, 2, code)
	require.Contains(t, e, "unknown charts command 'xyzzy'")
	require.NotContains(t, e, "did you mean")
}

func TestChartsMain_HelpPrintsUsage(t *testing.T) {
	for _, arg := range [][]string{{}, {"help"}, {"--help"}, {"-h"}} {
		var code int
		o, _ := capture(t, func() { code = chartsMain(arg) })
		require.Equal(t, 0, code)
		require.Contains(t, o, "Usage: swarmcli charts <command> [options]")
	}
}

// The README blocks show commands as an operator types them; the usage text
// shows them bare. Both come from the same rows.
func TestRenderCommands_PrefixIsApplied(t *testing.T) {
	block := renderCommands("swarmcli charts ")
	require.Contains(t, block, "  swarmcli charts install <release> <chart>")
	require.NotContains(t, renderCommands(""), "swarmcli charts install")
}

// A command with no arguments must not render trailing whitespace before its
// summary column, and every line must align on one column across all groups.
func TestRenderCommands_AlignsOneColumn(t *testing.T) {
	var col int
	for _, l := range strings.Split(strings.TrimRight(renderCommands(""), "\n"), "\n") {
		if !strings.HasPrefix(l, "  ") {
			continue // group heading
		}
		require.NotContains(t, l, "  \n")
		i := strings.Index(l[2:], "  ")
		require.Greater(t, i, 0, "no summary column in %q", l)
		i += 2
		for l[i] == ' ' {
			i++
		}
		if col == 0 {
			col = i
		}
		require.Equal(t, col, i, "summary column moves at %q", l)
	}
	require.NotZero(t, col)
}
