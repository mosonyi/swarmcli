// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

const topUsage = `swarmcli — keyboard-driven Docker Swarm manager

Run with no arguments to launch the interactive TUI.

Commands:
  charts      Manage charts (Helm-like package manager)
  version     Print the swarmcli version
  help        Show this help

Run "swarmcli charts help" for chart commands.
`

// binaryVersion is the version this binary reports for itself, recorded by
// Dispatch so diagnostics can name it. It is not necessarily the version of the
// chart engine the binary embeds (charts.EngineVersion), which is what chart
// compatibility is actually checked against.
var binaryVersion string

// Dispatch runs a non-interactive command and returns a process exit code. It
// is invoked from main when the binary is given arguments; an empty args slice
// is never passed (main launches the TUI in that case). version is the build
// version string for `swarmcli version`.
func Dispatch(args []string, version string) int {
	binaryVersion = version
	switch args[0] {
	case "charts":
		return chartsMain(args[1:])
	case "version", "--version", "-v":
		outln(version)
		return 0
	case "help", "--help", "-h":
		out(topUsage)
		return 0
	default:
		return usageErr("unknown command " + quote(args[0]) + "\n\n" + topUsage)
	}
}

func quote(s string) string { return "\"" + s + "\"" }
