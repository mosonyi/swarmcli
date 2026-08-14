// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

// Package chartsview is the read-only browser for chart releases.
//
// The package directory is views/charts but the package is chartsview, so that
// the release engine it reads from keeps the plain `charts` identifier.
package chartsview

import swarmlog "github.com/Eldara-Tech/swarmcli/utils/log"

const ViewName = "charts"

// readOnlyHint is always on screen. The view deliberately cannot change a
// release, and an operator should learn where that lives without having to
// press a key and be told no.
const readOnlyHint = "Read-only · manage releases with `swarmcli charts install|upgrade|rollback|uninstall`"

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
