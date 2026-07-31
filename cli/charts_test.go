// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/Eldara-Tech/swarmcli/charts"
)

// capture redirects stdout/stderr to separate buffers for the duration of fn
// and returns (stdout, stderr) so callers can assert on the exact stream.
func capture(t *testing.T, fn func()) (string, string) {
	t.Helper()
	var outBuf, errBuf bytes.Buffer
	origOut, origErr := stdout, stderr
	stdout, stderr = &outBuf, &errBuf
	defer func() { stdout, stderr = origOut, origErr }()
	fn()
	return outBuf.String(), errBuf.String()
}

func TestDispatchVersion(t *testing.T) {
	var code int
	o, _ := capture(t, func() { code = Dispatch([]string{"version"}, "1.2.3") })
	require.Equal(t, 0, code)
	require.Equal(t, "1.2.3", strings.TrimSpace(o))
}

func TestDispatchUnknownCommand(t *testing.T) {
	var code int
	capture(t, func() { code = Dispatch([]string{"frobnicate"}, "dev") })
	require.Equal(t, 2, code)
}

func TestChartsTemplateLocalDir(t *testing.T) {
	var code int
	o, _ := capture(t, func() {
		code = Dispatch([]string{"charts", "template", "my-demo", "../charts/testdata/demo", "--set", "replicas=3", "--set", "image.tag=v9"}, "dev")
	})
	require.Equal(t, 0, code)
	require.Contains(t, o, "traefik:v9")
	require.Contains(t, o, "replicas: 3")
	require.Contains(t, o, "com.swarmcli.release: my-demo")
}

func TestChartsTemplateSchemaRejection(t *testing.T) {
	var code int
	_, errOut := capture(t, func() {
		code = Dispatch([]string{"charts", "template", "x", "../charts/testdata/demo", "--set", "replicas=0"}, "dev")
	})
	require.Equal(t, 1, code)
	require.Contains(t, errOut, "schema validation")
}

func TestChartsTemplateUnknownFlag(t *testing.T) {
	var code int
	capture(t, func() {
		code = Dispatch([]string{"charts", "template", "x", "../charts/testdata/demo", "--bogus"}, "dev")
	})
	require.Equal(t, 2, code)
}

func TestChartsPruneTooManyArgs(t *testing.T) {
	var code int
	_, errOut := capture(t, func() {
		code = Dispatch([]string{"charts", "prune", "rel-a", "rel-b"}, "dev")
	})
	require.Equal(t, 2, code)
	require.Contains(t, errOut, "charts prune [release]")
}

func TestChartsPruneUnknownFlag(t *testing.T) {
	var code int
	capture(t, func() {
		code = Dispatch([]string{"charts", "prune", "--bogus"}, "dev")
	})
	require.Equal(t, 2, code)
}

func TestChartsShowValues(t *testing.T) {
	var code int
	o, _ := capture(t, func() {
		code = Dispatch([]string{"charts", "show", "values", "../charts/testdata/demo"}, "dev")
	})
	require.Equal(t, 0, code)
	require.Contains(t, o, "replicas: 1")
}

// show values must print values.yaml verbatim — its comments and key order are
// documentation and must survive (re-marshalling the parsed map would drop both).
func TestChartsShowValuesPreservesComments(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "templates"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "Chart.yaml"), []byte("name: demo\nversion: 1.0.0\n"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "templates", "stack.yaml"), []byte("services: {}\n"), 0o644))
	raw := "# leading comment\nreplicas: 1  # inline comment\n"
	require.NoError(t, os.WriteFile(filepath.Join(dir, "values.yaml"), []byte(raw), 0o644))

	var code int
	o, _ := capture(t, func() {
		code = Dispatch([]string{"charts", "show", "values", dir}, "dev")
	})
	require.Equal(t, 0, code)
	require.Equal(t, raw, o)
}

func TestParseArgs(t *testing.T) {
	pos, f, err := parseArgs([]string{"rel", "repo/chart", "-f", "a.yaml", "--values", "b.yaml", "--set", "x=1", "--dry-run", "--timeout", "10m"})
	require.NoError(t, err)
	require.Equal(t, []string{"rel", "repo/chart"}, pos)
	require.Equal(t, []string{"a.yaml", "b.yaml"}, f.values)
	require.Equal(t, []string{"x=1"}, f.sets)
	require.True(t, f.dryRun)
	require.Equal(t, "10m0s", f.timeout.String())
}

func TestParseArgsInlineValue(t *testing.T) {
	_, f, err := parseArgs([]string{"--set=a=1", "--version=2.0.0"})
	require.NoError(t, err)
	require.Equal(t, []string{"a=1"}, f.sets)
	require.Equal(t, "2.0.0", f.version)
}

func TestParseArgsRequirements(t *testing.T) {
	pos, f, err := parseArgs([]string{"rel", "repo/chart", "--requirements"})
	require.NoError(t, err)
	require.Equal(t, []string{"rel", "repo/chart"}, pos)
	require.True(t, f.requirements)
}

// --debug was parsed into a field nothing ever read, so it was accepted and did
// nothing — the exact "reads as if it worked" shape #451 is about. It is gone,
// which means it now falls through to the unknown-flag error like any other
// name the CLI does not know.
func TestParseArgsRejectsDebug(t *testing.T) {
	_, _, err := parseArgs([]string{"--debug"})
	require.EqualError(t, err, `unknown flag "--debug"`)
}

func TestParseIntRejectsGarbage(t *testing.T) {
	n, err := parseInt("3")
	require.NoError(t, err)
	require.Equal(t, 3, n)
	for _, bad := range []string{"2x", "-1", "", " ", "1.5", "0x10"} {
		_, err := parseInt(bad)
		require.Errorf(t, err, "expected %q to be rejected", bad)
	}
}

// --- apply (Docker-free paths only; the engine is covered in charts/) ---

func TestChartsApplyRequiresExactlyOneFile(t *testing.T) {
	var code int
	capture(t, func() { code = chartsApply(nil) })
	require.Equal(t, 2, code, "no release file")

	capture(t, func() { code = chartsApply([]string{"-f", "a.yaml", "-f", "b.yaml"}) })
	require.Equal(t, 2, code, "two release files")
}

func TestChartsApplyMissingFile(t *testing.T) {
	var code int
	_, e := capture(t, func() { code = chartsApply([]string{"-f", filepath.Join(t.TempDir(), "nope.yaml")}) })
	require.Equal(t, 1, code)
	require.Contains(t, e, "nope.yaml")
}

// A typo in a file an automated updater rewrites must fail loudly and name the key.
//
//nolint:misspell // "verison" is a deliberate typo — it is what the test rejects.
func TestChartsApplyRejectsUnknownKey(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rel.yaml")
	require.NoError(t, os.WriteFile(path, []byte("releases:\n  - name: a\n    chart: r/c\n    verison: \"1\"\n"), 0o600))

	var code int
	_, e := capture(t, func() { code = chartsApply([]string{"-f", path}) })
	require.Equal(t, 1, code)
	require.Contains(t, e, "verison")
}

// The charts flag set is global, so every subcommand parses every flag. apply must
// REJECT the ones it does not honour rather than silently ignoring them: its whole
// contract is that the file is the only source of truth.
func TestChartsApplyRejectsUnsupportedFlags(t *testing.T) {
	path := filepath.Join(t.TempDir(), "rel.yaml")
	require.NoError(t, os.WriteFile(path, []byte("releases:\n  - {name: a, chart: r/c, version: \"1\"}\n"), 0o600))

	for _, flag := range [][]string{
		{"--set", "a=1"},
		{"--version", "1.0.0"},
		{"--reuse-values"},
		{"--install"},
		{"--purge-volumes"},
	} {
		t.Run(flag[0], func(t *testing.T) {
			var code int
			_, e := capture(t, func() { code = chartsApply(append([]string{"-f", path}, flag...)) })
			require.Equal(t, 2, code)
			require.Contains(t, e, flag[0])
			require.Contains(t, e, "only source of truth")
		})
	}
}

func TestPrintPlan(t *testing.T) {
	plan := &charts.Plan{Releases: []charts.ReleasePlan{
		{Name: "hello", Ref: "r/whoami", Action: charts.ActionInstall, ToVersion: "0.1.8"},
		{Name: "edge", Ref: "r/traefik", Action: charts.ActionUpgrade, FromVersion: "0.1.0", ToVersion: "0.1.1"},
		{Name: "cache", Ref: "r/redis", Action: charts.ActionUnchanged, FromVersion: "0.1.1", ToVersion: "0.1.1"},
	}}
	o, _ := capture(t, func() { printPlan(plan, false) })

	require.Contains(t, o, "RELEASE")
	require.Contains(t, o, "hello")
	require.Contains(t, o, "install")
	require.Contains(t, o, "upgrade")
	require.Contains(t, o, "unchanged")
	// An install has no prior version; it must render as "-", not empty.
	require.Regexp(t, `hello\s+r/whoami\s+-\s+0\.1\.8\s+install`, o)
	require.Contains(t, o, "1 to install, 1 to upgrade, 1 unchanged")
}

// --diff shows a manifest diff for changed releases and skips unchanged ones.
func TestPrintPlanWithDiff(t *testing.T) {
	plan := &charts.Plan{Releases: []charts.ReleasePlan{
		{
			Name: "edge", Ref: "r/traefik", Action: charts.ActionUpgrade,
			FromVersion: "0.1.0", ToVersion: "0.1.1",
			CurrentManifest: "services:\n  a: 1\n", Manifest: "services:\n  a: 2\n",
		},
		{Name: "cache", Ref: "r/redis", Action: charts.ActionUnchanged, ToVersion: "0.1.1"},
	}}
	o, _ := capture(t, func() { printPlan(plan, true) })

	require.Contains(t, o, "--- edge (upgrade) ---")
	require.NotContains(t, o, "--- cache", "an unchanged release has nothing to diff")
}

// apply never deletes; it reports what it left alone, with the command to remove it.
func TestReportUnmanaged(t *testing.T) {
	o, _ := capture(t, func() {
		reportUnmanaged(&charts.Plan{Unmanaged: []string{"legacy", "scratch"}})
	})
	require.Contains(t, o, "2 release(s) exist on this swarm but are not in the release file")
	require.Contains(t, o, "swarmcli charts uninstall legacy")
	require.Contains(t, o, "swarmcli charts uninstall scratch")

	// Nothing unmanaged must print nothing at all — not a stray header on every
	// clean apply.
	o, _ = capture(t, func() { reportUnmanaged(&charts.Plan{}) })
	require.Empty(t, o)
}

// An orphan is reported separately from an unmanaged release, and in its own
// words: the stamp says this file installed it, so "not in the release file"
// would understate what is actually known about it.
func TestReportOrphaned(t *testing.T) {
	o, _ := capture(t, func() {
		reportOrphaned(&charts.Plan{Owner: "apply/prod", Orphaned: []string{"gone"}})
	})
	require.Contains(t, o, "1 release(s) were installed by this release file but are no longer declared in it")
	require.Contains(t, o, "swarmcli charts uninstall gone")

	o, _ = capture(t, func() { reportOrphaned(&charts.Plan{}) })
	require.Empty(t, o)
}

// The two lists always travel together. They answer one question — what does the
// swarm hold that this apply did not deploy — from two directions, and every
// exit path routes through the pair rather than picking one. Reporting only one
// of them is what the dry-run path used to do, by reporting neither.
func TestReportUnclaimedCoversBothLists(t *testing.T) {
	o, _ := capture(t, func() {
		reportUnclaimed(&charts.Plan{
			Owner:     "apply/prod",
			Orphaned:  []string{"gone"},
			Unmanaged: []string{"legacy"},
		})
	})
	require.Contains(t, o, "no longer declared in it")
	require.Contains(t, o, "not in the release file")

	o, _ = capture(t, func() { reportUnclaimed(&charts.Plan{}) })
	require.Empty(t, o, "a clean apply prints no headers at all")
}

// The store is https-only by default, and this env var is the whole of the
// compatibility story for anyone already pointing swarmcli at an internal
// plaintext registry — so assert the wiring, not just the parsing.
func TestNewStoreWiresThePlaintextOptOut(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())

	s, code := newStore()
	require.Equal(t, -1, code)
	require.False(t, s.AllowPlaintext, "https-only unless the operator says otherwise")

	t.Setenv(charts.AllowPlaintextEnv, "1")
	s, code = newStore()
	require.Equal(t, -1, code)
	require.True(t, s.AllowPlaintext)

	// Anything that is not a truthy boolean leaves the default in place, rather
	// than any non-empty value opting out by accident.
	t.Setenv(charts.AllowPlaintextEnv, "maybe")
	s, _ = newStore()
	require.False(t, s.AllowPlaintext)
}
