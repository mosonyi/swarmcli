// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package cli

import "github.com/Eldara-Tech/swarmcli/v2/charts"

// topUsageHead lists what this package itself serves. Anything a build
// embedding this module registered is rendered after it — see topUsage.
const topUsageHead = `swarmcli — keyboard-driven Docker Swarm manager

Run with no arguments to launch the interactive TUI.

Commands:
  charts      Manage charts (Helm-like package manager)
  version     Print the swarmcli version
  help        Show this help
`

const topUsageTail = `
Run "swarmcli charts help" for chart commands.
`

// topUsage is the help this binary can honestly print: the built-in verbs
// plus whatever a build embedding this module registered. A build of this
// repository alone registers nothing and prints exactly what it printed
// before the seam existed.
func topUsage() string {
	return topUsageHead + externalUsage() + topUsageTail
}

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
//
// The chart engine is reported beside the binary's own version because the two
// can legitimately differ and only one of them governs chart compatibility: the
// engine is this module's code, so a build that embeds swarmcli as a dependency
// carries whichever engine it pinned, regardless of its own tag.
//
// `unstamped` rather than silence when the ldflag did not take. An unstamped
// engine makes CheckCompat report CompatUnknown, so every chart's declared
// swarmcliVersion floor is admitted unchecked — a release that has done that
// looks entirely normal otherwise, and this is the one place it is visible.
func versionLine(version string) string {
	engine := charts.EngineVersion()
	if engine == "" {
		engine = "unstamped"
	}

	switch buildEdition {
	case "ce":
		return version + " (oss build, chart engine " + engine + ")"
	case "be":
		return version + " (business build, chart engine " + engine + ")"
	default:
		return version + " (chart engine " + engine + ")"
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
		out(topUsage())
		return 0
	}

	// A verb this binary does not implement itself. Checked after the
	// built-ins, and RegisterCommand refuses to shadow one, so the
	// vocabulary of this package cannot be changed from outside it — only
	// extended.
	if c, ok := externalCommands[args[0]]; ok {
		return c.run(args[1:])
	}
	return usageErr("unknown command " + quote(args[0]) + "\n\n" + topUsage())
}

func quote(s string) string { return "\"" + s + "\"" }
