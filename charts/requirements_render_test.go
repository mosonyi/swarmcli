// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// End-to-end: rendering requirements.yaml with the same values as the manifest
// makes an operator-chosen network name pass the pre-flight, where requirements
// resolved against defaults (the pre-fix behaviour) would reject it.
func TestRenderedRequirementsMatchManifestNetwork(t *testing.T) {
	ch := &Chart{
		Metadata: Chartfile{Name: "kc", Version: "0.1.0"},
		Values:   map[string]any{"database": map[string]any{"network": "keycloak-db-net"}},
		Templates: map[string]string{"templates/stack.yaml": "" +
			"services:\n" +
			"  app:\n" +
			"    image: nginx:1\n" +
			"    networks:\n" +
			"      - {{ .Values.database.network }}\n" +
			"networks:\n" +
			"  {{ .Values.database.network }}:\n" +
			"    external: true\n"},
		RequirementsRaw: []byte("networks:\n  - name: \"{{ .Values.database.network }}\"\n    autoCreate: false\n"),
	}

	// Operator points the DB at the mariadb chart's overlay.
	vals, err := MergeValues(ch.Values, nil, []string{"database.network=mariadb-net"})
	require.NoError(t, err)
	ctx := RenderContext{Values: vals, Release: ReleaseMeta{Name: "kc", Namespace: "kc", Revision: 1}, Chart: ChartMeta{Name: "kc", Version: "0.1.0"}}
	manifest, err := Render(ch, ctx)
	require.NoError(t, err)

	fb := newFakeBackend()
	fb.networkScopes["mariadb-net"] = "swarm" // the overlay already exists (validate-only)
	e := testEngine(fb)

	// Fixed: requirements rendered with the same override declare mariadb-net.
	req, err := RenderRequirements(ch, ctx)
	require.NoError(t, err)
	_, err = e.ensureExternalNetworks(context.Background(), manifest, req)
	require.NoError(t, err, "resolved requirement must match the resolved manifest network")

	// Pre-fix analogue: requirements resolved against defaults still say
	// keycloak-db-net, so the overridden manifest fails the contract.
	def, err := RenderRequirements(ch, RenderContext{Values: ch.Values})
	require.NoError(t, err)
	_, err = e.ensureExternalNetworks(context.Background(), manifest, def)
	require.Error(t, err)
	require.Contains(t, err.Error(), "not declared in requirements.yaml")
	require.Contains(t, err.Error(), "mariadb-net")
}

// A templated, quoted requirement value is resolved against the same values that
// produce the manifest, so an operator-chosen network name is what the pre-flight
// validates against. Non-name fields (autoCreate) and sibling resources survive.
func TestRenderRequirementsResolvesValues(t *testing.T) {
	ch := &Chart{
		Templates: map[string]string{},
		RequirementsRaw: []byte(
			"networks:\n" +
				"  - name: \"{{ .Values.database.network }}\"\n" +
				"    autoCreate: false\n" +
				"secrets:\n" +
				"  - name: app_pw\n"),
	}
	ctx := RenderContext{Values: map[string]any{
		"database": map[string]any{"network": "shared-db-net"},
	}}

	req, err := RenderRequirements(ch, ctx)
	require.NoError(t, err)
	require.NotNil(t, req)
	require.Len(t, req.Networks, 1)
	require.Equal(t, "shared-db-net", req.Networks[0].Name)
	require.False(t, *req.Networks[0].AutoCreate) // preserved through render+parse
	require.Len(t, req.Secrets, 1)
	require.Equal(t, "app_pw", req.Secrets[0].Name)
}

// A chart without requirements.yaml yields (nil, nil) — the manifest-driven
// fallback, unchanged.
func TestRenderRequirementsNilWhenAbsent(t *testing.T) {
	ch := &Chart{Templates: map[string]string{}}
	req, err := RenderRequirements(ch, RenderContext{})
	require.NoError(t, err)
	require.Nil(t, req)
}

// A requirements.yaml with no template directives renders byte-identically, so
// existing charts are unaffected.
func TestRenderRequirementsStaticUnchanged(t *testing.T) {
	ch := &Chart{
		Templates:       map[string]string{},
		RequirementsRaw: []byte("networks:\n  - name: fixed-net\n"),
	}
	req, err := RenderRequirements(ch, RenderContext{Values: map[string]any{}})
	require.NoError(t, err)
	require.Len(t, req.Networks, 1)
	require.Equal(t, "fixed-net", req.Networks[0].Name)
}

// A template error in requirements.yaml surfaces as an error, not a silent miss.
func TestRenderRequirementsBadTemplate(t *testing.T) {
	ch := &Chart{
		Templates:       map[string]string{},
		RequirementsRaw: []byte("networks:\n  - name: \"{{ .Values.missing | \"\n"),
	}
	_, err := RenderRequirements(ch, RenderContext{Values: map[string]any{}})
	require.Error(t, err)
}
