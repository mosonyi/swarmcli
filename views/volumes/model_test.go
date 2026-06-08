// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package volumesview

import (
	"context"
	"testing"
	"time"

	"swarmcli/docker"
	inspectview "swarmcli/views/inspect"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/volume"
	"github.com/stretchr/testify/require"
)

// --- mocks ---

type mockVolumeOps struct {
	listVolumesFn   func(ctx context.Context) ([]docker.VolumeInfo, error)
	inspectVolumeFn func(ctx context.Context, name string) (volume.Volume, error)
}

func (m *mockVolumeOps) ListVolumes(ctx context.Context) ([]docker.VolumeInfo, error) {
	return m.listVolumesFn(ctx)
}
func (m *mockVolumeOps) InspectVolume(ctx context.Context, name string) (volume.Volume, error) {
	return m.inspectVolumeFn(ctx, name)
}

func noopVolumeOps() *mockVolumeOps {
	return &mockVolumeOps{
		listVolumesFn: func(_ context.Context) ([]docker.VolumeInfo, error) {
			return nil, nil
		},
		inspectVolumeFn: func(_ context.Context, _ string) (volume.Volume, error) {
			return volume.Volume{}, nil
		},
	}
}

// --- helpers ---

func key(s string) tea.KeyMsg {
	switch s {
	case "enter":
		return tea.KeyMsg{Type: tea.KeyEnter}
	case "esc":
		return tea.KeyMsg{Type: tea.KeyEsc}
	case "up":
		return tea.KeyMsg{Type: tea.KeyUp}
	case "down":
		return tea.KeyMsg{Type: tea.KeyDown}
	case "left":
		return tea.KeyMsg{Type: tea.KeyLeft}
	case "right":
		return tea.KeyMsg{Type: tea.KeyRight}
	}
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}
}

func runCmd(cmd tea.Cmd) tea.Msg {
	if cmd == nil {
		return nil
	}
	return cmd()
}

func testModel(opts ...func(*Model)) *Model {
	m := New(80, 24)
	m.deps = docker.Deps{Volumes: noopVolumeOps()}
	for _, o := range opts {
		o(m)
	}
	return m
}

func fakeVolumes(names ...string) []volumeItem {
	now := time.Now()
	items := make([]volumeItem, len(names))
	for i, name := range names {
		items[i] = volumeItem{
			Name:       name,
			Stack:      "mystack",
			Driver:     "local",
			Mountpoint: "/var/lib/docker/volumes/" + name + "/_data",
			Created:    now,
			Host:       "node-1",
		}
	}
	return items
}

func loadVolumes(m *Model, items []volumeItem) {
	m.Update(VolumesLoadedMsg{Volumes: items})
}

// --- Tests ---

func TestNew(t *testing.T) {
	m := New(80, 24)
	require.Equal(t, 80, m.width)
	require.Equal(t, 24, m.height)
	require.Equal(t, stateLoading, m.state)
	require.True(t, m.visible)
	require.Equal(t, SortByName, m.sortField)
	require.True(t, m.sortAscending)
}

func TestName(t *testing.T) {
	require.Equal(t, "volumes", testModel().Name())
}

func TestCapturesInput(t *testing.T) {
	m := testModel()
	require.False(t, m.CapturesInput())
	m.errorDialogActive = true
	require.True(t, m.CapturesInput())
}

func TestHasActiveFilter(t *testing.T) {
	m := testModel()
	require.False(t, m.HasActiveFilter())
	m.ApplySearchQuery("x")
	require.True(t, m.HasActiveFilter())
}

func TestHasErrors(t *testing.T) {
	require.False(t, testModel().HasErrors())
}

func TestOnEnter_SetsVisible(t *testing.T) {
	m := testModel()
	m.visible = false
	cmd := m.OnEnter()
	require.True(t, m.visible)
	require.True(t, m.resetCursorOnNextLoad)
	require.NotNil(t, cmd)
}

func TestOnExit_ClearsVisible(t *testing.T) {
	m := testModel()
	m.OnExit()
	require.False(t, m.visible)
}

func TestLoaded_SetsReadyAndSorts(t *testing.T) {
	m := testModel()
	loadVolumes(m, fakeVolumes("beta", "alpha", "gamma"))
	require.Equal(t, stateReady, m.state)
	require.Len(t, m.volumesList.Filtered, 3)
	// default sort: name ascending
	require.Equal(t, "alpha", m.volumesList.Filtered[0].Name)
	require.Equal(t, "gamma", m.volumesList.Filtered[2].Name)
}

func TestSort_ToggleNameDescending(t *testing.T) {
	m := testModel()
	loadVolumes(m, fakeVolumes("beta", "alpha", "gamma"))
	m.Update(key("N")) // already SortByName -> toggles to descending
	require.False(t, m.sortAscending)
	require.Equal(t, "gamma", m.volumesList.Filtered[0].Name)
}

func TestFilter_NarrowsResults(t *testing.T) {
	m := testModel()
	loadVolumes(m, fakeVolumes("alpha", "beta", "gamma"))
	m.ApplySearchQuery("alp")
	require.Len(t, m.volumesList.Filtered, 1)
	require.Equal(t, "alpha", m.volumesList.Filtered[0].Name)

	m.ClearSearchQuery()
	require.Len(t, m.volumesList.Filtered, 3)
}

func TestInspect_NavigatesToInspectView(t *testing.T) {
	m := testModel(func(m *Model) {
		m.deps = docker.Deps{Volumes: &mockVolumeOps{
			listVolumesFn: func(_ context.Context) ([]docker.VolumeInfo, error) { return nil, nil },
			inspectVolumeFn: func(_ context.Context, name string) (volume.Volume, error) {
				return volume.Volume{Name: name, Driver: "local"}, nil
			},
		}}
	})
	loadVolumes(m, fakeVolumes("vol1"))
	msg := runCmd(m.Update(key("i")))
	nav, ok := msg.(view.NavigateToMsg)
	require.True(t, ok)
	require.Equal(t, inspectview.ViewName, nav.ViewName)
	payload, ok := nav.Payload.(map[string]any)
	require.True(t, ok)
	require.Contains(t, payload["title"], "vol1")
}

func TestInspect_NoVolumes_NoOp(t *testing.T) {
	m := testModel()
	require.Nil(t, m.Update(key("i")))
}
