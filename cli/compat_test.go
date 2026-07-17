// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

import (
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"swarmcli/charts"
)

// withInteractive forces the "is there a human to ask" answer for one test, so
// both branches are reachable without a pty.
func withInteractive(t *testing.T, v bool) {
	t.Helper()
	prev := isInteractive
	isInteractive = func() bool { return v }
	t.Cleanup(func() { isInteractive = prev })
}

// withStdin feeds a canned answer to the confirm prompt.
func withStdin(t *testing.T, s string) {
	t.Helper()
	prev := stdin
	stdin = strings.NewReader(s)
	t.Cleanup(func() { stdin = prev })
}

func withBinaryVersion(t *testing.T, v string) {
	t.Helper()
	prev := binaryVersion
	binaryVersion = v
	t.Cleanup(func() { binaryVersion = prev })
}

// failingReader fails the test if anything reads it.
type failingReader struct{ t *testing.T }

func (r failingReader) Read([]byte) (int, error) {
	r.t.Error("stdin was read under a policy that forbids prompting")
	return 0, io.EOF
}

func incompatibleFinding() charts.CompatFinding {
	return charts.CompatFinding{
		Chart:    "traefik 0.2.0",
		Required: ">= 1.13.0",
		Engine:   "1.12.0",
		Status:   charts.CompatIncompatible,
	}
}

func TestApplyCompatProceedsWhenCompatible(t *testing.T) {
	for _, pol := range []compatPolicy{compatWarn, compatEnforce, compatEnforceNoPrompt} {
		var code int
		_, e := capture(t, func() {
			code = applyCompat(charts.CompatFinding{Status: charts.CompatOK}, pol, false)
		})
		require.Equal(t, -1, code)
		require.Empty(t, e, "a compatible chart must say nothing")
	}
}

func TestApplyCompatUnknownNeverBlocks(t *testing.T) {
	t.Run("silent when the chart declared nothing", func(t *testing.T) {
		var code int
		_, e := capture(t, func() {
			code = applyCompat(charts.CompatFinding{Status: charts.CompatUnknown}, compatEnforce, false)
		})
		require.Equal(t, -1, code)
		require.Empty(t, e)
	})

	// The author expected the constraint to be honoured; silence would hide that
	// it was not.
	t.Run("warns when a declared constraint was unusable", func(t *testing.T) {
		f := charts.CompatFinding{
			Chart:  "demo 0.1.0",
			Status: charts.CompatUnknown,
			Reason: `swarmcliVersion "nope" is not a valid SemVer constraint`,
		}
		var code int
		_, e := capture(t, func() { code = applyCompat(f, compatEnforce, false) })
		require.Equal(t, -1, code)
		require.Contains(t, e, "demo 0.1.0")
		require.Contains(t, e, "compatibility was not checked")
	})
}

func TestApplyCompatWarnNeverBlocks(t *testing.T) {
	withInteractive(t, false)
	var code int
	_, e := capture(t, func() { code = applyCompat(incompatibleFinding(), compatWarn, false) })
	require.Equal(t, -1, code)
	require.Contains(t, e, "requires swarmcli >= 1.13.0")
}

// The CI contract: no terminal means abort, never hang.
func TestApplyCompatEnforceNonInteractiveAborts(t *testing.T) {
	withInteractive(t, false)
	withBinaryVersion(t, "1.12.0")
	for _, pol := range []compatPolicy{compatEnforce, compatEnforceNoPrompt} {
		var code int
		_, e := capture(t, func() { code = applyCompat(incompatibleFinding(), pol, false) })
		require.Equal(t, 1, code)
		require.Contains(t, e, "requires swarmcli >= 1.13.0")
		require.Contains(t, e, "--skip-compat-check", "the abort must name its own escape hatch")
	}
}

func TestApplyCompatSkipProceeds(t *testing.T) {
	withInteractive(t, false)
	for _, pol := range []compatPolicy{compatWarn, compatEnforce, compatEnforceNoPrompt} {
		var code int
		_, e := capture(t, func() { code = applyCompat(incompatibleFinding(), pol, true) })
		require.Equal(t, -1, code)
		require.Contains(t, e, "--skip-compat-check", "a forced run must still leave a trace")
	}
}

func TestApplyCompatEnforceInteractivePrompt(t *testing.T) {
	for _, tc := range []struct {
		name  string
		input string
		want  int
	}{
		{name: "y proceeds", input: "y\n", want: -1},
		{name: "yes proceeds", input: "yes\n", want: -1},
		{name: "uppercase Y proceeds", input: "Y\n", want: -1},
		{name: "y without a newline proceeds", input: "y", want: -1},
		{name: "n aborts", input: "n\n", want: 1},
		// Default-deny: the safe answer is the one that takes no keystroke.
		{name: "a blank line aborts", input: "\n", want: 1},
		{name: "EOF aborts", input: "", want: 1},
		{name: "anything else aborts", input: "maybe\n", want: 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			withInteractive(t, true)
			withStdin(t, tc.input)
			var code int
			_, e := capture(t, func() { code = applyCompat(incompatibleFinding(), compatEnforce, false) })
			require.Equal(t, tc.want, code)
			require.Contains(t, e, "Continue anyway?")
		})
	}
}

// apply is meant to run unattended: a terminal happening to be attached must not
// make it block on stdin.
func TestApplyCompatEnforceNoPromptNeverReadsStdin(t *testing.T) {
	withInteractive(t, true)
	prev := stdin
	stdin = failingReader{t: t}
	t.Cleanup(func() { stdin = prev })

	var code int
	_, e := capture(t, func() { code = applyCompat(incompatibleFinding(), compatEnforceNoPrompt, false) })
	require.Equal(t, 1, code)
	require.NotContains(t, e, "Continue anyway?")
}
