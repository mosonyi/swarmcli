// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/Eldara-Tech/swarmcli/charts"
)

// compatPolicy is what a subcommand does with a chart that declares a newer
// chart engine than this build provides. Detection lives in the charts package;
// this is the half that decides what it costs.
type compatPolicy int

const (
	// compatWarn reports and carries on. It is for the read-only verbs: they
	// change nothing, and `charts show` in particular is how an operator finds
	// out which version a chart wants — refusing to run it would withhold the
	// answer to the question being asked.
	compatWarn compatPolicy = iota
	// compatEnforce refuses, asking first when there is someone to ask.
	compatEnforce
	// compatEnforceNoPrompt refuses without ever reading stdin. It is for apply,
	// whose contract is to run unattended: prompting merely because it happened
	// to be launched from a terminal would make it hang in the one place that
	// must never hang.
	compatEnforceNoPrompt
)

// isInteractive reports whether there is a human to prompt — both stdin and
// stdout must be a terminal. A var so tests can drive both branches without a
// pty. Using os.ModeCharDevice keeps this dependency-free: nothing else in the
// chart CLI has any reason to know about terminals.
var isInteractive = func() bool {
	return isCharDevice(os.Stdin) && isCharDevice(os.Stdout)
}

func isCharDevice(f *os.File) bool {
	fi, err := f.Stat()
	return err == nil && fi.Mode()&os.ModeCharDevice != 0
}

// stdin is a package var for the same reason stdout/stderr are: so tests can
// feed it.
var stdin io.Reader = os.Stdin

// applyCompat applies pol to a compatibility finding. It returns an exit code
// >= 0 when the caller must stop, or -1 to carry on.
func applyCompat(f charts.CompatFinding, pol compatPolicy, skip bool) int {
	switch f.Status {
	case charts.CompatOK:
		return -1
	case charts.CompatUnknown:
		// A chart that declared nothing is the common case and says nothing. One
		// that declared something unusable is worth a word: its author expected
		// the constraint to be honoured, and silence would hide that it was not.
		if f.Reason != "" {
			errf("chart %s: %s; compatibility was not checked\n", f.Chart, f.Reason)
		}
		return -1
	}

	msg := f.Message(binaryVersion)
	switch {
	case skip:
		errf("%s; continuing anyway (--skip-compat-check)\n", msg)
		return -1
	case pol == compatWarn:
		errf("%s\n", msg)
		return -1
	case pol == compatEnforce && isInteractive():
		errf("%s\n", msg)
		if confirm("Continue anyway?") {
			return -1
		}
		return 1
	default:
		return fail(fmt.Errorf("%s\n  upgrade swarmcli, or re-run with --skip-compat-check to try anyway", msg))
	}
}

// confirm asks a yes/no question on stderr and reads one line from stdin.
//
// Anything but an explicit yes — a blank line, EOF, a read error — is no. The
// prompt exists to stop an install that is expected to break, so the safe answer
// must be the one that takes no keystroke.
func confirm(question string) bool {
	errf("\n%s [y/N] ", question)
	line, err := bufio.NewReader(stdin).ReadString('\n')
	if err != nil && line == "" {
		errf("\n")
		return false
	}
	switch strings.ToLower(strings.TrimSpace(line)) {
	case "y", "yes":
		return true
	default:
		return false
	}
}
