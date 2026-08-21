// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

// Package chartsview is the read-only browser for chart releases.
//
// The package directory is views/charts but the package is chartsview, so that
// the release engine it reads from keeps the plain `charts` identifier.
package chartsview

import swarmlog "github.com/Eldara-Tech/swarmcli/v2/utils/log"

const ViewName = "charts"

// readOnlyHint is always on screen. The view deliberately cannot change a
// release, and an operator should learn where that lives without having to
// press a key and be told no.
// It is kept short enough to survive an 80-column frame: the full command list
// overflowed the border and read as a truncated sentence, and `?` carries the
// verbs anyway.
const readOnlyHint = "Read-only · change releases with `swarmcli charts` (? for how)"

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("views", "charts")
}

// displayOrDash renders an em-dash for empty optional fields.
func displayOrDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
