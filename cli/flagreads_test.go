// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

import (
	"go/ast"
	"go/parser"
	"go/token"
	"sort"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// fieldFlag maps a field of the flags struct to the flag that sets it. The
// struct is the parser's output and the table is the CLI's contract; this is
// the only place the two vocabularies meet.
var fieldFlag = map[string]string{
	"values":          "--values",
	"sets":            "--set",
	"setFiles":        "--set-file",
	"version":         "--version",
	"dryRun":          "--dry-run",
	"wait":            "--wait",
	"requirements":    "--requirements",
	"purge":           "--purge-volumes",
	"install":         "--install",
	"reuseValues":     "--reuse-values",
	"diff":            "--diff",
	"revision":        "--revision",
	"timeout":         "--timeout",
	"historyMax":      "--history-max",
	"skipCompatCheck": "--skip-compat-check",
	"noRepoUpdate":    "--no-repo-update",
	"forVersion":      "--for-version",
	"resolveImage":    "--resolve-image",
}

// flagPlumbing are the functions that inspect the whole flags value rather than
// reading it as a command does. Following them would credit every command with
// every flag.
var flagPlumbing = map[string]bool{"parse": true, "parseArgs": true, "reject": true}

// flagReads reports, per function in this package, which flags it honours: the
// f.<field> reads in its own body plus those of every helper it hands its own f
// to (newStore, loadChart, prepare — the reason a handler's body understates
// what it reads).
func flagReads(t *testing.T) map[string]map[string]bool {
	t.Helper()

	fset := token.NewFileSet()
	files, err := parser.ParseDir(fset, ".", nil, 0)
	require.NoError(t, err)
	pkg, ok := files["cli"]
	require.True(t, ok, "cli package did not parse")

	direct := map[string]map[string]bool{}  // func -> flags read in its own body
	callees := map[string]map[string]bool{} // func -> funcs it passes its own f to

	for _, file := range pkg.Files {
		for _, decl := range file.Decls {
			fn, ok := decl.(*ast.FuncDecl)
			if !ok || flagPlumbing[fn.Name.Name] {
				continue
			}
			name := fn.Name.Name
			direct[name] = map[string]bool{}
			callees[name] = map[string]bool{}

			ast.Inspect(fn, func(n ast.Node) bool {
				switch v := n.(type) {
				case *ast.SelectorExpr:
					// f.dryRun — a read of the parsed flag value.
					if id, ok := v.X.(*ast.Ident); ok && id.Name == "f" {
						if flag, ok := fieldFlag[v.Sel.Name]; ok {
							direct[name][flag] = true
						}
					}
				case *ast.CallExpr:
					// helper(…, f, …) — the helper reads on this command's behalf.
					// A call passing a fresh flags literal instead (chartsRepo's
					// newStore(flags{…})) is deliberately not followed: those
					// values come from the code, not from the command line.
					id, ok := v.Fun.(*ast.Ident)
					if !ok {
						return true
					}
					for _, a := range v.Args {
						if ai, ok := a.(*ast.Ident); ok && ai.Name == "f" {
							callees[name][id.Name] = true
						}
					}
				}
				return true
			})
		}
	}

	// Close over the helper chain: prepare calls loadChart calls newStore.
	out := map[string]map[string]bool{}
	for name := range direct {
		seen := map[string]bool{}
		var walk func(string)
		walk = func(n string) {
			if seen[n] {
				return
			}
			seen[n] = true
			for f := range direct[n] {
				if out[name] == nil {
					out[name] = map[string]bool{}
				}
				out[name][f] = true
			}
			for c := range callees[n] {
				walk(c)
			}
		}
		walk(name)
		if out[name] == nil {
			out[name] = map[string]bool{}
		}
	}
	return out
}

// The allow-lists were written by reading the handlers; this reads them back
// out of the syntax tree and compares. Without it, a row that lists too few
// flags rejects an invocation that used to work — and every table-driven test
// in this package would agree with the wrong table, because they all iterate
// it.
func TestFlagAllowListsMatchWhatHandlersRead(t *testing.T) {
	reads := flagReads(t)

	for _, c := range chartsCommands {
		t.Run(c.Name, func(t *testing.T) {
			fn := handlerName(t, c)
			got, ok := reads[fn]
			require.True(t, ok, "handler %s not found in the syntax tree", fn)

			want := map[string]bool{}
			for _, f := range c.Flags {
				want[f] = true
			}
			require.Equal(t, sortedKeys(want), sortedKeys(got),
				"%s's allow-list and what %s reads have diverged", c.Name, fn)
		})
	}
}

// handlerName recovers the function name behind a row's Run, which is what
// links the table to the syntax tree.
func handlerName(t *testing.T, c chartsCmd) string {
	t.Helper()
	return "charts" + strings.ToUpper(c.Name[:1]) + c.Name[1:]
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
