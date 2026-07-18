// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

// engineVersion is the chart-engine feature level, stamped at build time with
//
//	-X swarmcli/charts.engineVersion=<version>
//
// It names the version of THIS module, which is not necessarily the version the
// surrounding binary reports for itself: the chart engine is this module's code,
// so a downstream binary that embeds it carries whichever engine it pinned,
// regardless of its own tag. Such a build stamps this with the swarmcli version
// it pins; a plain swarmcli build stamps it with its own.
//
// It is empty in an unstamped build (`go build`, `go run`, dev), and CheckCompat
// then reports CompatUnknown rather than blocking — not knowing the engine
// version is not evidence that a chart is incompatible with it.
var engineVersion = ""

// EngineVersion reports the chart-engine version this binary embeds, or the
// empty string for an unstamped build.
func EngineVersion() string { return engineVersion }
