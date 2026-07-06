// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package servicesview

import (
	"swarmcli/features"
	"swarmcli/views/view"
)

// serviceHealthFeature gates the per-container HEALTH/PORTS enrichment. The base
// build never enables it, so CE always shows the upsell footer hint; an
// extension build enabling it (via the license reconciler) both lights up the
// HEALTH column and suppresses the hint — mirroring the volumes
// "volumes-all-nodes" pattern. Matches license.FeatureServiceHealth.
const serviceHealthFeature = "service-health"

// serviceHealthUpsellHint advertises the Business Edition capability in the
// services footer when the feature is unavailable (CE or unlicensed).
var serviceHealthUpsellHint = "Per-container health & ports across nodes is a Business Edition feature: " + view.BELandingURL

// healthFooterHint returns the extra services-view footer line, or "":
//   - feature off (CE / unlicensed): the BE upsell hint.
//   - feature on, but the extension reports health is unreachable on the current
//     context (via view.ServicesHealthHint): that context note.
//   - feature on and health reachable: "" — the HEALTH column carries the info.
func healthFooterHint() string {
	if features.IsEnabled(serviceHealthFeature) {
		if view.ServicesHealthHint != nil {
			return view.ServicesHealthHint()
		}
		return ""
	}
	return serviceHealthUpsellHint
}
