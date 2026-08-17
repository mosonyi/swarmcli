// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"testing"

	"github.com/Eldara-Tech/swarmcli/charts"

	"github.com/stretchr/testify/require"
)

// The cell drops the half of the stamp that repeats the row, and spells the
// controller's namespace as the product that writes it.
func TestOwnerCell(t *testing.T) {
	for _, tc := range []struct {
		name, stamp, release, want string
	}{
		{"controller", "cd/default/whoami:release/whoami", "whoami", "swarmcli-cd/default/whoami"},
		{"controller, pre-controller-id stamp", "cd/whoami:release/whoami", "whoami", "swarmcli-cd/whoami"},
		{"release file", "apply/prod-swarm:release/hello", "hello", "apply/prod-swarm"},
		{"unowned", "", "hello", "—"},
		// Not evidence of owning this release, so it is not summarised as if it
		// were — the operator gets what is actually stored.
		{"names another release", "cd/prod/edge:release/other", "api", "cd/prod/edge:release/other"},
		{"another kind", "apply/prod:stack/hello", "hello", "apply/prod:stack/hello"},
		{"unparseable", "who-knows", "hello", "who-knows"},
		// An id naming no application is left as it is rather than dressed up
		// as a controller id it is not.
		{"bare prefix", "cd/:release/api", "api", "cd/"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, ownerCell(tc.stamp, tc.release))
		})
	}
}

// The two halves are the engine's, not this view's invention: what ownerCell
// drops is exactly what the stamp encoding puts there, so a change to either
// side of it fails here rather than silently rendering a different string.
func TestOwnerCellDropsTheResourceHalfTheEngineWrote(t *testing.T) {
	stamp := charts.OwnerRef{ID: "cd/default/whoami", Kind: charts.OwnerKindRelease, Name: "whoami"}.String()
	require.Equal(t, "cd/default/whoami:release/whoami", stamp)
	require.Equal(t, "swarmcli-cd/default/whoami", ownerCell(stamp, "whoami"))
}
