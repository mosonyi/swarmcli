// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package portsview

import (
	"testing"

	"swarmcli/docker"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	m := New(80, 20)
	require.NotNil(t, m)
	require.Equal(t, 80, m.width)
	require.Equal(t, 20, m.height)
	require.False(t, m.ready)
	require.Equal(t, ViewName, m.Name())
	require.NotEmpty(t, m.ShortHelpItems())
	require.False(t, m.HasErrors())
}

func TestUpdate_WindowSize(t *testing.T) {
	m := New(80, 20)
	msg := tea.WindowSizeMsg{Width: 100, Height: 40}
	cmd := m.Update(msg)
	require.Nil(t, cmd)
	require.Equal(t, 100, m.width)
	require.Equal(t, 40, m.height)
	require.True(t, m.ready)

	// Content should be set in viewport
	content := m.viewport.View()
	require.Contains(t, content, "2377")
	require.Contains(t, content, "7946")
	require.Contains(t, content, "4789")
}

func TestUpdate_Keys(t *testing.T) {
	m := New(80, 20)
	m.ready = true

	// Test Esc key returns GoBackMsg
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	cmd := m.Update(msg)
	require.NotNil(t, cmd)
	
	res := cmd()
	require.IsType(t, view.GoBackMsg{}, res)
}

func TestView(t *testing.T) {
	m := New(80, 40)
	// View is empty before ready
	require.Empty(t, m.View())

	// Once ready, View is not empty and has frame elements
	m.ready = true
	m.updateViewport()
	out := m.View()
	require.NotEmpty(t, out)
	require.Contains(t, out, "Required Swarm Ports")
	require.Contains(t, out, "2377")
	require.Contains(t, out, "7946")
	require.Contains(t, out, "4789")
}

type mockSnapshotOps struct {
	docker.SnapshotOps
	snap *docker.SwarmSnapshot
}

func (m mockSnapshotOps) GetSnapshot() *docker.SwarmSnapshot {
	return m.snap
}

func TestGetDiagnosticStatus(t *testing.T) {
	// 1. Nil snapshot
	m := New(80, 40)
	require.Empty(t, m.getDiagnosticStatus())

	// Set up dependencies
	m.deps = docker.Deps{}
	
	// 2. Mock snap with empty nodes
	mockOps := mockSnapshotOps{
		snap: &docker.SwarmSnapshot{},
	}
	m.deps.Snapshot = mockOps
	require.Contains(t, m.getDiagnosticStatus(), "No nodes found")

	// Helper to build node entries
	// 3. Healthy single node manager
	snap := &docker.SwarmSnapshot{
		Nodes: []swarm.Node{
			{
				ID: "node1",
				Spec: swarm.NodeSpec{
					Role: swarm.NodeRoleManager,
				},
				Status: swarm.NodeStatus{
					State: swarm.NodeStateReady,
				},
				ManagerStatus: &swarm.ManagerStatus{
					Leader: true,
				},
			},
		},
	}
	mockOps.snap = snap
	m.deps.Snapshot = mockOps
	
	diag := m.getDiagnosticStatus()
	require.Contains(t, diag, "HEALTHY / ONLINE")
	require.Contains(t, diag, "TCP 2377")
	require.Contains(t, diag, "TCP/UDP 7946")
	require.Contains(t, diag, "UDP 4789")

	// 4. Degraded gossip network (offline worker)
	snap.Nodes = append(snap.Nodes, swarm.Node{
		ID: "node2",
		Spec: swarm.NodeSpec{
			Role: swarm.NodeRoleWorker,
		},
		Status: swarm.NodeStatus{
			State: swarm.NodeStateDown,
		},
	})
	diag = m.getDiagnosticStatus()
	require.Contains(t, diag, "DEGRADED / DISRUPTED")
	require.Contains(t, diag, "offline")

	// 5. Unreachable manager status
	snap.Nodes = []swarm.Node{
		{
			ID: "node1",
			Spec: swarm.NodeSpec{
				Role: swarm.NodeRoleManager,
			},
			Status: swarm.NodeStatus{
				State: swarm.NodeStateReady,
			},
			ManagerStatus: &swarm.ManagerStatus{
				Reachability: swarm.ReachabilityUnreachable,
			},
		},
	}
	diag = m.getDiagnosticStatus()
	require.Contains(t, diag, "UNREACHABLE / BLOCKED")
}
