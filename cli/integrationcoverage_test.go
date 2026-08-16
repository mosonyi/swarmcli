// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

import (
	"go/ast"
	"go/parser"
	"go/token"
	"strconv"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// integrationSuiteDir is the package that drives this CLI against a real swarm.
// It is read rather than imported: its tests are behind the integration build
// tag and need a Docker daemon, while this one needs neither.
const integrationSuiteDir = "../integration-tests/charts"

// dispatchFuncs are the two ways that suite invokes the CLI: cli.Dispatch, and
// the dispatchOK helper wrapping it.
var dispatchFuncs = map[string]bool{"Dispatch": true, "dispatchOK": true}

// integrationInvocations reports, per charts subcommand, which flags the
// integration suite passes it, read out of that suite's syntax tree.
//
// Only the literal arguments are read: a release name, a chart directory and a
// temp path are variables, and none of them is a flag. An invocation therefore
// counts when its "charts" and command words are literals, which every one of
// them writes out.
func integrationInvocations(t *testing.T) map[string]map[string]bool {
	t.Helper()

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, integrationSuiteDir, nil, 0)
	require.NoError(t, err, "the integration suite did not parse")
	require.NotEmpty(t, pkgs, "%s holds no Go package", integrationSuiteDir)

	used := map[string]map[string]bool{}
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			ast.Inspect(file, func(n ast.Node) bool {
				call, ok := n.(*ast.CallExpr)
				if !ok || !isDispatchCall(call) {
					return true
				}
				record(used, literalArgs(call))
				return true
			})
		}
	}
	return used
}

// isDispatchCall reports whether call runs the CLI.
func isDispatchCall(call *ast.CallExpr) bool {
	switch fn := call.Fun.(type) {
	case *ast.Ident:
		return dispatchFuncs[fn.Name]
	case *ast.SelectorExpr:
		return dispatchFuncs[fn.Sel.Name]
	}
	return false
}

// literalArgs returns the string literals inside call, in source order. The
// arguments may be spread across the call (dispatchOK) or wrapped in a slice
// literal (cli.Dispatch), so the whole subtree is walked rather than the
// argument list.
func literalArgs(call *ast.CallExpr) []string {
	var out []string
	ast.Inspect(call, func(n ast.Node) bool {
		lit, ok := n.(*ast.BasicLit)
		if !ok || lit.Kind != token.STRING {
			return true
		}
		if s, err := strconv.Unquote(lit.Value); err == nil {
			out = append(out, s)
		}
		return true
	})
	return out
}

// record credits the command named in argv with every flag argv passes it.
func record(used map[string]map[string]bool, argv []string) {
	var i int
	for i < len(argv) && argv[i] != "charts" {
		i++
	}
	if i+1 >= len(argv) {
		return
	}
	c, ok := lookupCommand(argv[i+1])
	if !ok {
		return
	}
	if used[c.Name] == nil {
		used[c.Name] = map[string]bool{}
	}
	for _, a := range argv[i+2:] {
		if !strings.HasPrefix(a, "-") {
			continue
		}
		name, _, _ := splitFlag(a)
		if long, ok := canonicalFlag[name]; ok {
			name = long
		}
		used[c.Name][name] = true
	}
}

// Every command in the table must be run against a real swarm, with every flag
// it accepts.
//
// The allow-lists are what this layer is for: a flag outside a row exits 2
// instead of being parsed and dropped, so a row that is too narrow rejects an
// invocation that used to work. TestFlagAllowListsMatchWhatHandlersRead proves
// each row matches its handler; only running the command proves the row is
// right. This reads the invocations out of the integration suite rather than
// keeping a list of what is covered, because a hand-kept list of coverage is
// the first thing to stop being true — a flag added to a row would otherwise
// leave every test here green and nothing running it.
func TestIntegrationSuiteExercisesEveryCommandAndFlag(t *testing.T) {
	used := integrationInvocations(t)
	require.NotEmpty(t, used, "no CLI invocation was found in %s — this test would pass on an empty suite", integrationSuiteDir)

	for _, c := range chartsCommands {
		t.Run(c.Name, func(t *testing.T) {
			require.Contains(t, used, c.Name,
				"nothing in %s runs `swarmcli charts %s` against a real swarm", integrationSuiteDir, c.Name)
			for _, f := range c.Flags {
				require.True(t, used[c.Name][f],
					"`swarmcli charts %s` is never run with %s in %s — add it to an invocation there",
					c.Name, f, integrationSuiteDir)
			}
		})
	}
}
