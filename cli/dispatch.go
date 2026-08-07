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

// buildEdition names which artefact this binary is, for `swarmcli version`.
//
// One tag publishes two of them under the same command name: this repository's
// own build, and a build from the private extension wrapper carrying licensed
// code that is inert without a licence. The version string is identical for
// both, so it cannot tell them apart — see docs/editions.md.
//
// This is a property of the *build*, deliberately distinct from the edition
// label the TUI shows, which follows live licence state and reads "Community
// Edition" on an unlicensed extension build. That is the right answer for the
// header and the wrong one here: it cannot distinguish "this binary has no
// licensed code" from "this binary has no licence".
//
// Empty prints nothing, which is what every build did before this existed —
// so a binary whose main has not been taught to set it stays silent rather
// than claiming to be the artefact it is not.
var buildEdition string

// SetEdition records which artefact this build is (called from main).
func SetEdition(e string) { buildEdition = e }

// versionLine renders `swarmcli version`.
func versionLine(version string) string {
	switch buildEdition {
	case "ce":
		return version + " (oss build)"
	case "be":
		return version + " (business build)"
	default:
		return version
	}
}

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
		outln(versionLine(version))
		return 0
	case "help", "--help", "-h":
		out(topUsage)
		return 0
	default:
		return usageErr("unknown command " + quote(args[0]) + "\n\n" + topUsage)
	}
}

func quote(s string) string { return "\"" + s + "\"" }
