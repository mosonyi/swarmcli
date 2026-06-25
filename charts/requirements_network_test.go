// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package charts

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

// requirements.yaml drives the network pre-flight: an autoCreate:true network is
// created with the declared driver and attachability.
func TestEnsureExternalNetworksAutoCreateHonorsDriver(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	req := &Requirements{Networks: []NetworkRequirement{
		{Name: "traefik-public", Driver: "overlay", Attachable: boolPtr(false), AutoCreate: boolPtr(true)},
	}}
	created, err := e.ensureExternalNetworks(context.Background(), extNetManifest, req)
	require.NoError(t, err)
	require.Equal(t, []string{"traefik-public"}, created)
	require.Equal(t, createdNet{driver: "overlay", attachable: false}, fb.createdNets["traefik-public"])
}

// An autoCreate:false network is validated, never created: a missing one is a
// hard error carrying the declared description, and nothing is created.
func TestEnsureExternalNetworksValidateOnlyMissing(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	req := &Requirements{Networks: []NetworkRequirement{
		{Name: "traefik-public", Driver: "overlay", Attachable: boolPtr(true), AutoCreate: boolPtr(false), Description: "Traefik ingress"},
	}}
	created, err := e.ensureExternalNetworks(context.Background(), extNetManifest, req)
	require.Error(t, err)
	require.Empty(t, created)
	require.NotContains(t, fb.networkScopes, "traefik-public", "validate-only network must not be created")
	require.Contains(t, err.Error(), "traefik-public: does not exist (Traefik ingress)")
}

// A pre-existing swarm-scoped network satisfies an autoCreate:false requirement.
func TestEnsureExternalNetworksValidateOnlyPresent(t *testing.T) {
	fb := newFakeBackend()
	fb.networkScopes["traefik-public"] = "swarm"
	e := testEngine(fb)
	req := &Requirements{Networks: []NetworkRequirement{
		{Name: "traefik-public", Driver: "overlay", Attachable: boolPtr(true), AutoCreate: boolPtr(false)},
	}}
	created, err := e.ensureExternalNetworks(context.Background(), extNetManifest, req)
	require.NoError(t, err)
	require.Empty(t, created)
}

// When requirements.yaml is present it is authoritative: an external network the
// manifest references but the file does not declare is a contract error, and no
// network is created.
func TestEnsureExternalNetworksUndeclaredIsContractError(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	req := &Requirements{} // present but declares no networks
	created, err := e.ensureExternalNetworks(context.Background(), extNetManifest, req)
	require.Error(t, err)
	require.Empty(t, created)
	require.NotContains(t, fb.networkScopes, "traefik-public")
	require.Contains(t, err.Error(), "network(s) the manifest declares external are not declared in requirements.yaml")
	require.Contains(t, err.Error(), "traefik-public")
}

// Install records the networks it auto-created on the revision so uninstall can
// report them, and uninstall returns them as orphaned while leaving them in
// place.
func TestInstallRecordsAndUninstallReportsManagedNetworks(t *testing.T) {
	ctx := context.Background()
	fb := newFakeBackend()
	e := testEngine(fb)
	req := &Requirements{Networks: []NetworkRequirement{
		{Name: "traefik-public", Driver: "overlay", Attachable: boolPtr(true), AutoCreate: boolPtr(true)},
	}}

	rel, err := e.Install(ctx, "demo", ReleaseChart{Name: "demo", Version: "1"}, nil, extNetManifest,
		InstallOptions{Requirements: req})
	require.NoError(t, err)
	require.Equal(t, []string{"traefik-public"}, rel.ManagedNetworks)

	res, err := e.Uninstall(ctx, "demo", false)
	require.NoError(t, err)
	require.Equal(t, []string{"traefik-public"}, res.OrphanedNetworks)
	require.Contains(t, fb.networkScopes, "traefik-public", "uninstall must not remove the network")
}

// A managed network the operator removed out-of-band is not reported as
// orphaned.
func TestUninstallSkipsManagedNetworkAlreadyGone(t *testing.T) {
	ctx := context.Background()
	fb := newFakeBackend()
	e := testEngine(fb)
	req := &Requirements{Networks: []NetworkRequirement{
		{Name: "traefik-public", Driver: "overlay", Attachable: boolPtr(true), AutoCreate: boolPtr(true)},
	}}
	_, err := e.Install(ctx, "demo", ReleaseChart{Name: "demo", Version: "1"}, nil, extNetManifest,
		InstallOptions{Requirements: req})
	require.NoError(t, err)

	delete(fb.networkScopes, "traefik-public") // operator removed it themselves

	res, err := e.Uninstall(ctx, "demo", false)
	require.NoError(t, err)
	require.Empty(t, res.OrphanedNetworks)
}

// Without a requirements.yaml the network pre-flight keeps its historical
// behaviour: a missing external network is auto-created as an attachable
// overlay.
func TestEnsureExternalNetworksFallbackWithoutRequirements(t *testing.T) {
	fb := newFakeBackend()
	e := testEngine(fb)
	created, err := e.ensureExternalNetworks(context.Background(), extNetManifest, nil)
	require.NoError(t, err)
	require.Equal(t, []string{"traefik-public"}, created)
	require.Equal(t, createdNet{driver: "overlay", attachable: true}, fb.createdNets["traefik-public"])
}
