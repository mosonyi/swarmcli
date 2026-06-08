// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package volumesview

import (
	"context"
	"testing"

	"swarmcli/docker"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/require"
)

// sentinelMsg is returned by stub actions so tests can assert they fired.
type sentinelMsg struct{ arg string }

// registerStubAction registers a temporary action capturing its argument.
func registerStubAction(t *testing.T, name string) *string {
	t.Helper()
	var got string
	view.RegisterAction(name, func(arg string) tea.Cmd {
		got = arg
		return func() tea.Msg { return sentinelMsg{arg: arg} }
	})
	t.Cleanup(func() { view.UnregisterActionForTest(name) })
	return &got
}

func TestKey_Create_DispatchesAction(t *testing.T) {
	got := registerStubAction(t, "volume-create")
	m := testModel()
	loadVolumes(m, fakeVolumes("vol1"))

	msg := runCmd(m.Update(key("c")))
	require.IsType(t, sentinelMsg{}, msg)
	require.Equal(t, "", *got) // create is selection-independent
	require.False(t, m.errorDialogActive)
}

func TestKey_Browse_PassesEncodedNodeAndName(t *testing.T) {
	got := registerStubAction(t, "volume-browse")
	m := testModel()
	items := fakeVolumes("vol1")
	items[0].NodeID = "node-id-123"
	loadVolumes(m, items)

	msg := runCmd(m.Update(key("b")))
	require.IsType(t, sentinelMsg{}, msg)
	parts := view.DecodeRef(*got)
	require.Equal(t, []string{"node-id-123", "vol1", "node-1"}, parts)
}

func TestKey_Delete_PassesEncodedNodeAndName(t *testing.T) {
	got := registerStubAction(t, "volume-delete")
	m := testModel()
	items := fakeVolumes("vol1")
	items[0].NodeID = "node-id-123"
	loadVolumes(m, items)

	msg := runCmd(m.Update(tea.KeyMsg{Type: tea.KeyCtrlD}))
	require.IsType(t, sentinelMsg{}, msg)
	require.Equal(t, []string{"node-id-123", "vol1", "node-1"}, view.DecodeRef(*got))
}

func TestKey_Action_Unregistered_ShowsBEDialog(t *testing.T) {
	m := testModel()
	loadVolumes(m, fakeVolumes("vol1"))

	cmd := m.Update(key("c")) // no "volume-create" registered
	require.Nil(t, cmd)
	require.True(t, m.errorDialogActive)
	require.ErrorContains(t, m.err, "Business Edition")
}

func TestKey_Browse_NoSelection_NoOp(t *testing.T) {
	registerStubAction(t, "volume-browse")
	m := testModel() // empty list
	require.Nil(t, m.Update(key("b")))
	require.False(t, m.errorDialogActive)
}

func TestPartialList_ShowsBannerNotErrorDialog(t *testing.T) {
	m := testModel()
	m.Update(VolumesLoadedMsg{Volumes: fakeVolumes("vol1", "vol2"), Warn: "2 nodes unreachable"})

	require.Equal(t, stateReady, m.state)
	require.False(t, m.errorDialogActive)
	require.Len(t, m.volumesList.Filtered, 2)
	require.Contains(t, m.renderVolumesFooter(), "2 nodes unreachable")

	// A subsequent full load clears the banner.
	m.Update(VolumesLoadedMsg{Volumes: fakeVolumes("vol1", "vol2")})
	require.Empty(t, m.partialWarn)
	require.NotContains(t, m.renderVolumesFooter(), "unreachable")
}

func TestLoadVolumesCmd_MapsPartialListErrorToWarn(t *testing.T) {
	m := testModel(func(m *Model) {
		m.deps = docker.Deps{Volumes: &mockVolumeOps{
			listVolumesFn: func(_ context.Context) ([]docker.VolumeInfo, error) {
				return []docker.VolumeInfo{{Name: "vol1", NodeID: "n1"}},
					&docker.PartialListError{NodeErrors: map[string]string{"n2": "timeout"}}
			},
		}}
	})

	msg := runCmd(m.loadVolumesCmd())
	loaded, ok := msg.(VolumesLoadedMsg)
	require.True(t, ok)
	require.NoError(t, loaded.Err)
	require.Equal(t, "1 node unreachable", loaded.Warn)
	require.Len(t, loaded.Volumes, 1)
	require.Equal(t, "n1", loaded.Volumes[0].NodeID)
}
