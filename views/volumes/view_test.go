// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package volumesview

import (
	"strings"
	"testing"

	"swarmcli/features"
	"swarmcli/views/view"

	"github.com/stretchr/testify/require"
)

func TestView_LoadingState(t *testing.T) {
	m := testModel()
	m.state = stateLoading
	require.Contains(t, m.View(), "Docker Volumes")
}

func TestView_ReadyState_ShowsVolumes(t *testing.T) {
	m := testModel()
	loadVolumes(m, fakeVolumes("alpha", "beta"))
	m.volumesList.Viewport.Width = 100
	m.volumesList.Viewport.Height = 20
	out := m.View()
	require.Contains(t, out, "Docker Volumes (2)")
	require.Contains(t, out, "alpha")
	require.Contains(t, out, "beta")
}

func TestHeader_ShowsAllColumns(t *testing.T) {
	m := testModel()
	m.volumesList.Viewport.Width = 120
	loadVolumes(m, fakeVolumes("alpha"))
	header := m.volumesList.RenderHeader()
	for _, col := range []string{"NAME", "STACK", "DRIVER", "MOUNT POINT", "CREATED", "HOST"} {
		require.Contains(t, header, col)
	}
}

func TestFooter_BusinessEditionHint(t *testing.T) {
	m := testModel()
	loadVolumes(m, fakeVolumes("vol1"))

	// Flag off (default): connected-node hint is shown.
	require.Contains(t, m.renderVolumesFooter(), view.BELandingURL)

	// Flag on: hint suppressed.
	features.Enable(allNodesFeature)
	t.Cleanup(func() { features.Disable(allNodesFeature) })
	require.NotContains(t, m.renderVolumesFooter(), view.BELandingURL)
}

func TestDescription_RendersOptionalDashes(t *testing.T) {
	it := volumeItem{Name: "v", Driver: "local"}
	desc := it.Description()
	require.True(t, strings.Contains(desc, "—"), "empty stack/host should render an em-dash")
}
