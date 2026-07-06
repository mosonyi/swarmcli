// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package servicesview

import (
	"testing"

	"swarmcli/features"
	"swarmcli/views/view"

	"github.com/stretchr/testify/require"
)

func TestHealthFooterHint(t *testing.T) {
	// Flag off (CE / unlicensed): the upsell hint with the BE landing URL.
	features.Disable(serviceHealthFeature)
	view.ServicesHealthHint = nil
	require.Contains(t, healthFooterHint(), view.BELandingURL)

	// Flag on, no BE note: nothing (the HEALTH column carries the info).
	features.Enable(serviceHealthFeature)
	t.Cleanup(func() { features.Disable(serviceHealthFeature) })
	require.Equal(t, "", healthFooterHint())

	// Flag on, BE reports a context note: that note (and not the upsell).
	view.ServicesHealthHint = func() string { return "needs the managed context" }
	t.Cleanup(func() { view.ServicesHealthHint = nil })
	got := healthFooterHint()
	require.Equal(t, "needs the managed context", got)
	require.NotContains(t, got, view.BELandingURL)
}

func TestFrameFooter_AppendsHint(t *testing.T) {
	m := testModel()
	loadServices(m, fakeEntries("svc1"))

	// Flag off: the rendered footer carries the upsell hint.
	features.Disable(serviceHealthFeature)
	view.ServicesHealthHint = nil
	require.Contains(t, m.FrameFooter(), view.BELandingURL)

	// Flag on + a BE context note: the note shows, the upsell does not.
	features.Enable(serviceHealthFeature)
	t.Cleanup(func() { features.Disable(serviceHealthFeature) })
	view.ServicesHealthHint = func() string { return "needs the managed context" }
	t.Cleanup(func() { view.ServicesHealthHint = nil })
	foot := m.FrameFooter()
	require.Contains(t, foot, "needs the managed context")
	require.NotContains(t, foot, view.BELandingURL)

	// Flag on + no note (managed context, health works): neither hint.
	view.ServicesHealthHint = func() string { return "" }
	foot = m.FrameFooter()
	require.NotContains(t, foot, "needs the managed context")
	require.NotContains(t, foot, view.BELandingURL)
}
