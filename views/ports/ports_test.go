// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package portsview

import (
	"testing"
	"time"

	"swarmcli/docker"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/require"
)

// ── snapshot mock ─────────────────────────────────────────────────────────────

type mockSnapshotOps struct {
	docker.SnapshotOps
	snap *docker.SwarmSnapshot
}

func (m mockSnapshotOps) GetSnapshot() *docker.SwarmSnapshot            { return m.snap }
func (m mockSnapshotOps) TriggerRefreshIfNeeded()                       {}
func (m mockSnapshotOps) RefreshSnapshotAsync()                         {}
func (m mockSnapshotOps) SetSnapshot(s *docker.SwarmSnapshot)           {}
func (m mockSnapshotOps) InvalidateSnapshot()                           {}
func (m mockSnapshotOps) RefreshSnapshot() (*docker.SwarmSnapshot, error) { return m.snap, nil }
func (m mockSnapshotOps) GetOrRefreshSnapshot() (*docker.SwarmSnapshot, error) {
	return m.snap, nil
}

// ── basic construction ────────────────────────────────────────────────────────

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

// ── window-size msg ───────────────────────────────────────────────────────────

func TestUpdate_WindowSize(t *testing.T) {
	m := New(80, 20)
	cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	require.NotNil(t, cmd)        // batch (tick + probe launch)
	require.True(t, m.ready)
	require.Equal(t, 100, m.width)
	require.Equal(t, 40, m.height)

	content := m.viewport.View()
	require.Contains(t, content, "2377")
	require.Contains(t, content, "7946")
	require.Contains(t, content, "4789")
}

// ── esc key ───────────────────────────────────────────────────────────────────

func TestUpdate_EscKey(t *testing.T) {
	m := New(80, 20)
	m.ready = true
	cmd := m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	require.NotNil(t, cmd)
	require.IsType(t, view.GoBackMsg{}, cmd())
}

// ── r key (re-probe) ──────────────────────────────────────────────────────────

func TestUpdate_RKey_LaunchesProbe(t *testing.T) {
	m := New(80, 20)
	m.ready = true
	m.deps = docker.Deps{Snapshot: mockSnapshotOps{snap: &docker.SwarmSnapshot{}}}

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	require.NotNil(t, cmd, "r should return a probe command")

	// Running the command should produce a ProbeResultMsg.
	msg := cmd()
	require.IsType(t, ProbeResultMsg{}, msg)
}

func TestUpdate_RKey_NoopWhileProbing(t *testing.T) {
	m := New(80, 20)
	m.ready = true
	m.probing = true // simulate an in-flight probe

	cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
	require.Nil(t, cmd, "should not launch a second probe while one is in-flight")
}

// ── ProbeResultMsg delivery ───────────────────────────────────────────────────

func TestUpdate_ProbeResultMsg(t *testing.T) {
	m := New(80, 40)
	m.ready = true
	m.updateViewport()

	results := []docker.NodeProbeResult{
		{NodeID: "n1", Hostname: "node-1", Addr: "10.0.0.1", TCP2377: docker.PortOpen, TCP7946: docker.PortOpen, UDP7946: docker.PortOpen},
		{NodeID: "n2", Hostname: "node-2", Addr: "10.0.0.2", TCP2377: docker.PortFiltered, TCP7946: docker.PortOpen, UDP7946: docker.PortRefused},
	}

	cmd := m.Update(ProbeResultMsg{Results: results})
	require.Nil(t, cmd)

	stored := m.getProbeResults()
	require.Len(t, stored, 2)
	require.Equal(t, docker.PortOpen, stored[0].TCP2377)
	require.Equal(t, docker.PortFiltered, stored[1].TCP2377)
	require.Equal(t, docker.PortRefused, stored[1].UDP7946)

	// Viewport should now contain the node hostnames.
	content := m.viewport.View()
	require.Contains(t, content, "node-1")
	require.Contains(t, content, "node-2")
}

// ── TickMsg triggers refresh + probe ─────────────────────────────────────────

func TestUpdate_TickMsg_OldProbeTriggersNewProbe(t *testing.T) {
	m := New(80, 40)
	m.ready = true
	m.lastProbeAt = time.Now().Add(-ProbeInterval - time.Second) // expired
	m.deps = docker.Deps{Snapshot: mockSnapshotOps{snap: &docker.SwarmSnapshot{}}}

	cmd := m.Update(TickMsg(time.Now()))
	// Should return a batch that includes a probe launch.
	require.NotNil(t, cmd)
}

func TestUpdate_TickMsg_FreshProbeSkipsNewProbe(t *testing.T) {
	m := New(80, 40)
	m.ready = true
	m.lastProbeAt = time.Now()                                // just probed
	m.deps = docker.Deps{Snapshot: mockSnapshotOps{snap: &docker.SwarmSnapshot{}}}

	cmd := m.Update(TickMsg(time.Now()))
	// Should only return a tickCmd, not a full batch.
	require.NotNil(t, cmd)
}

// ── view output ───────────────────────────────────────────────────────────────

func TestView_BeforeReady(t *testing.T) {
	m := New(80, 40)
	require.Empty(t, m.View())
}

func TestView_AfterReady(t *testing.T) {
	m := New(80, 40)
	m.ready = true
	m.updateViewport()
	out := m.View()
	require.NotEmpty(t, out)
	require.Contains(t, out, "Required Swarm Ports")
	require.Contains(t, out, "2377")
	require.Contains(t, out, "7946")
	require.Contains(t, out, "4789")
}

func TestView_ShowsProbeResults(t *testing.T) {
	m := New(120, 60)
	m.ready = true
	m.probeResults = []docker.NodeProbeResult{
		{NodeID: "n1", Hostname: "manager-1", Addr: "192.168.1.10", NodeState: "ready", TCP2377: docker.PortOpen, TCP7946: docker.PortOpen, UDP7946: docker.PortOpen},
		{NodeID: "n2", Hostname: "worker-1", Addr: "192.168.1.11", NodeState: "ready", TCP2377: docker.PortFiltered, TCP7946: docker.PortRefused, UDP7946: docker.PortFiltered},
	}
	m.updateViewport()
	out := m.View()
	require.Contains(t, out, "manager-1")
	require.Contains(t, out, "worker-1")
	require.Contains(t, out, "OPEN")
	require.Contains(t, out, "FILTERED")
	require.Contains(t, out, "CLOSED")
}

// TestView_NodeDown_ShowsNodeDown verifies that a stopped node renders "NODE DOWN"
// across all port columns instead of the misleading FILTERED that a timed-out
// probe produces on a dead host.
func TestView_NodeDown_ShowsNodeDown(t *testing.T) {
	m := New(120, 60)
	m.ready = true
	m.probeResults = []docker.NodeProbeResult{
		// Running node — probe timed out on TCP 2377 (firewall), so FILTERED.
		{NodeID: "n1", Hostname: "manager-1", Addr: "192.168.1.10", NodeState: "ready", TCP2377: docker.PortFiltered, TCP7946: docker.PortOpen, UDP7946: docker.PortOpen},
		// Stopped node — probe also timed out, but Docker says it's "down".
		{NodeID: "n2", Hostname: "worker-stopped", Addr: "192.168.1.11", NodeState: "down", TCP2377: docker.PortFiltered, TCP7946: docker.PortFiltered, UDP7946: docker.PortOpen},
	}
	m.updateViewport()
	out := m.View()

	require.Contains(t, out, "NODE DOWN", "stopped node should show NODE DOWN, not FILTERED")
	require.Contains(t, out, "manager-1")
	require.Contains(t, out, "worker-stopped")
	// The running node's FILTERED should still appear (firewall signal).
	require.Contains(t, out, "FILTERED")
}

// ── getDiagnosticStatus ───────────────────────────────────────────────────────

func TestGetDiagnosticStatus_NilSnapshot(t *testing.T) {
	m := New(80, 40)
	require.Empty(t, m.getDiagnosticStatus())
}

func TestGetDiagnosticStatus_EmptyNodes(t *testing.T) {
	m := New(80, 40)
	m.deps = docker.Deps{Snapshot: mockSnapshotOps{snap: &docker.SwarmSnapshot{}}}
	require.Contains(t, m.getDiagnosticStatus(), "No nodes found")
}

func TestGetDiagnosticStatus_HealthyManager(t *testing.T) {
	m := New(80, 40)
	m.deps = docker.Deps{Snapshot: mockSnapshotOps{
		snap: &docker.SwarmSnapshot{
			Nodes: []swarm.Node{{
				ID:   "n1",
				Spec: swarm.NodeSpec{Role: swarm.NodeRoleManager},
				Status: swarm.NodeStatus{State: swarm.NodeStateReady},
				ManagerStatus: &swarm.ManagerStatus{Leader: true},
			}},
		},
	}}
	diag := m.getDiagnosticStatus()
	require.Contains(t, diag, "HEALTHY / ONLINE")
	require.Contains(t, diag, "TCP 2377")
	require.Contains(t, diag, "TCP/UDP 7946")
	require.Contains(t, diag, "UDP 4789")
}

func TestGetDiagnosticStatus_DegradedGossip(t *testing.T) {
	m := New(80, 40)
	m.deps = docker.Deps{Snapshot: mockSnapshotOps{
		snap: &docker.SwarmSnapshot{
			Nodes: []swarm.Node{
				{
					ID:     "n1",
					Spec:   swarm.NodeSpec{Role: swarm.NodeRoleManager},
					Status: swarm.NodeStatus{State: swarm.NodeStateReady},
					ManagerStatus: &swarm.ManagerStatus{Leader: true},
				},
				{
					ID:     "n2",
					Spec:   swarm.NodeSpec{Role: swarm.NodeRoleWorker},
					Status: swarm.NodeStatus{State: swarm.NodeStateDown},
				},
			},
		},
	}}
	diag := m.getDiagnosticStatus()
	require.Contains(t, diag, "DEGRADED / DISRUPTED")
}

func TestGetDiagnosticStatus_UnreachableManager(t *testing.T) {
	m := New(80, 40)
	m.deps = docker.Deps{Snapshot: mockSnapshotOps{
		snap: &docker.SwarmSnapshot{
			Nodes: []swarm.Node{{
				ID:   "n1",
				Spec: swarm.NodeSpec{Role: swarm.NodeRoleManager},
				Status: swarm.NodeStatus{State: swarm.NodeStateReady},
				ManagerStatus: &swarm.ManagerStatus{
					Reachability: swarm.ReachabilityUnreachable,
				},
			}},
		},
	}}
	diag := m.getDiagnosticStatus()
	require.Contains(t, diag, "UNREACHABLE / BLOCKED")
}

// ── PortStatus.String ─────────────────────────────────────────────────────────

func TestPortStatus_String(t *testing.T) {
	require.Equal(t, "OPEN", docker.PortOpen.String())
	require.Equal(t, "CLOSED", docker.PortRefused.String())
	require.Equal(t, "FILTERED", docker.PortFiltered.String())
	require.Equal(t, "UNKNOWN", docker.PortUnknown.String())
}
