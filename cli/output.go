// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

// Package cli is swarmcli's non-interactive command-line front-end. When the
// binary is invoked with arguments (e.g. `swarmcli charts install ...`) it runs
// a one-shot command that prints to stdout and exits, rather than launching the
// TUI. This is what makes charts usable from scripts, CI/CD and GitOps.
package cli

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"
)

// stdout/stderr are package vars so tests can capture output.
var (
	stdout io.Writer = os.Stdout
	stderr io.Writer = os.Stderr
)

// table writes a tab-separated table with a header row.
func table(headers []string, rows [][]string) {
	tw := tabwriter.NewWriter(stdout, 0, 0, 3, ' ', 0)
	printRow(tw, headers)
	for _, r := range rows {
		printRow(tw, r)
	}
	_ = tw.Flush()
}

func printRow(w io.Writer, cols []string) {
	for i, c := range cols {
		if i > 0 {
			_, _ = fmt.Fprint(w, "\t")
		}
		_, _ = fmt.Fprint(w, c)
	}
	_, _ = fmt.Fprintln(w)
}

func out(s string)                 { _, _ = fmt.Fprint(stdout, s) }
func outf(format string, a ...any) { _, _ = fmt.Fprintf(stdout, format, a...) }
func outln(a ...any)               { _, _ = fmt.Fprintln(stdout, a...) }
func errf(format string, a ...any) { _, _ = fmt.Fprintf(stderr, format, a...) }

// usageErr prints msg to stderr and returns exit code 2.
func usageErr(msg string) int {
	errf("Error: %s\n", msg)
	return 2
}

// fail prints an error to stderr and returns exit code 1.
func fail(err error) int {
	errf("Error: %v\n", err)
	return 1
}
