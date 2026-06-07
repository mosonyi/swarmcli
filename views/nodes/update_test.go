// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package nodesview

import (
	"context"
	"fmt"
	"testing"
	"time"

	"swarmcli/docker"
	"swarmcli/views/confirmdialog"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/swarm"
	"github.com/stretchr/testify/require"
)

// --- State machine tests ---

func TestUpdate_Msg_SetsContent(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.ready = true
	loadNodes(m, fakeNodes("node1", "node2"))
	require.Equal(t, 2, len(m.List.Items))
	require.True(t, m.Visible)
}

func TestUpdate_WindowSizeMsg(t *testing.T) {
	m := testModel()
	m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	require.Equal(t, 120, m.List.Viewport.Width)
	require.Equal(t, 40, m.List.Viewport.Height)
	require.True(t, m.ready)
}

func TestUpdate_TickMsg_Visible(t *testing.T) {
	m := testModel()
	m.Visible = true
	cmd := m.Update(TickMsg(time.Now()))
	require.NotNil(t, cmd)
}

func TestUpdate_TickMsg_NotVisible(t *testing.T) {
	m := testModel()
	m.Visible = false
	cmd := m.Update(TickMsg(time.Now()))
	require.NotNil(t, cmd) // still returns tickCmd
}

func TestUpdate_DemoteSuccessMsg(t *testing.T) {
	m := testModel()
	m.confirmDialog.Visible = true
	cmd := m.Update(DemoteSuccessMsg{})
	require.False(t, m.confirmDialog.Visible)
	require.NotNil(t, cmd) // reloads
}

func TestUpdate_DemoteErrorMsg(t *testing.T) {
	m := testModel()
	m.Update(DemoteErrorMsg{NodeID: "n1", Error: fmt.Errorf("failed")})
	require.True(t, m.confirmDialog.Visible)
	require.True(t, m.confirmDialog.ErrorMode)
	require.Contains(t, m.confirmDialog.Message, "failed")
}

func TestUpdate_PromoteSuccessMsg(t *testing.T) {
	m := testModel()
	m.confirmDialog.Visible = true
	cmd := m.Update(PromoteSuccessMsg{})
	require.False(t, m.confirmDialog.Visible)
	require.NotNil(t, cmd)
}

func TestUpdate_PromoteErrorMsg(t *testing.T) {
	m := testModel()
	m.Update(PromoteErrorMsg{NodeID: "n1", Error: fmt.Errorf("failed")})
	require.True(t, m.confirmDialog.Visible)
	require.True(t, m.confirmDialog.ErrorMode)
}

func TestUpdate_RemoveSuccessMsg(t *testing.T) {
	m := testModel()
	m.confirmDialog.Visible = true
	cmd := m.Update(RemoveSuccessMsg{})
	require.False(t, m.confirmDialog.Visible)
	require.NotNil(t, cmd)
}

func TestUpdate_RemoveErrorMsg(t *testing.T) {
	m := testModel()
	m.Update(RemoveErrorMsg{NodeID: "n1", Error: fmt.Errorf("failed")})
	require.True(t, m.confirmDialog.Visible)
	require.True(t, m.confirmDialog.ErrorMode)
}

func TestUpdate_SetAvailabilitySuccessMsg(t *testing.T) {
	m := testModel()
	m.availabilityDialog = true
	cmd := m.Update(SetAvailabilitySuccessMsg{})
	require.False(t, m.availabilityDialog)
	require.NotNil(t, cmd)
}

func TestUpdate_SetAvailabilityErrorMsg(t *testing.T) {
	m := testModel()
	m.Update(SetAvailabilityErrorMsg{NodeID: "n1", Error: fmt.Errorf("fail")})
	require.True(t, m.confirmDialog.Visible)
	require.True(t, m.confirmDialog.ErrorMode)
}

func TestUpdate_AddLabelSuccessMsg(t *testing.T) {
	m := testModel()
	m.labelInputDialog = true
	cmd := m.Update(AddLabelSuccessMsg{})
	require.False(t, m.labelInputDialog)
	require.NotNil(t, cmd)
}

func TestUpdate_AddLabelErrorMsg(t *testing.T) {
	m := testModel()
	m.Update(AddLabelErrorMsg{NodeID: "n1", Error: fmt.Errorf("fail")})
	require.True(t, m.confirmDialog.Visible)
	require.True(t, m.confirmDialog.ErrorMode)
}

func TestUpdate_RemoveLabelSuccessMsg(t *testing.T) {
	m := testModel()
	m.labelRemoveDialog = true
	cmd := m.Update(RemoveLabelSuccessMsg{})
	require.False(t, m.labelRemoveDialog)
	require.NotNil(t, cmd)
}

func TestUpdate_RemoveLabelErrorMsg(t *testing.T) {
	m := testModel()
	m.Update(RemoveLabelErrorMsg{NodeID: "n1", Error: fmt.Errorf("fail")})
	require.True(t, m.confirmDialog.Visible)
	require.True(t, m.confirmDialog.ErrorMode)
}

// --- Key routing tests ---

func TestKey_I_InspectsNode(t *testing.T) {
	m := testModel()
	loadNodes(m, fakeNodes("mynode"))
	cmd := m.Update(key("i"))
	msg := runCmd(cmd)
	nav, ok := msg.(view.NavigateToMsg)
	require.True(t, ok)
	require.Equal(t, "inspect", nav.ViewName)
}

func TestKey_P_ShowsServices(t *testing.T) {
	m := testModel()
	loadNodes(m, fakeNodes("mynode"))
	cmd := m.Update(key("p"))
	msg := runCmd(cmd)
	nav, ok := msg.(view.NavigateToMsg)
	require.True(t, ok)
	require.Equal(t, "services", nav.ViewName)
}

func TestKey_Help(t *testing.T) {
	m := testModel()
	loadNodes(m, fakeNodes("n1"))
	cmd := m.Update(key("?"))
	msg := runCmd(cmd)
	nav, ok := msg.(view.NavigateToMsg)
	require.True(t, ok)
	require.Equal(t, view.NameHelp, nav.ViewName)
}

func TestKey_CtrlP_ShowsPorts(t *testing.T) {
	m := testModel()
	loadNodes(m, fakeNodes("n1"))
	cmd := m.Update(key("ctrl+p"))
	msg := runCmd(cmd)
	nav, ok := msg.(view.NavigateToMsg)
	require.True(t, ok)
	require.Equal(t, view.NamePorts, nav.ViewName)
}

func TestKey_CtrlD_OpensRemoveConfirm(t *testing.T) {
	m := testModel()
	loadNodes(m, fakeNodes("mynode"))
	m.Update(key("ctrl+d"))
	require.True(t, m.confirmDialog.Visible)
	require.Contains(t, m.confirmDialog.Message, "mynode")
	require.Contains(t, m.confirmDialog.Message, "Remove")
}

func TestKey_CtrlT_DemoteManager(t *testing.T) {
	m := testModel()
	entries := fakeNodes("mgr")
	entries[0].Manager = true
	loadNodes(m, entries)
	m.Update(key("ctrl+t"))
	require.True(t, m.confirmDialog.Visible)
	require.Contains(t, m.confirmDialog.Message, "Demote")
}

func TestKey_CtrlT_NotManager_ShowsError(t *testing.T) {
	m := testModel()
	loadNodes(m, fakeNodes("worker"))
	m.Update(key("ctrl+t"))
	require.True(t, m.confirmDialog.Visible)
	require.True(t, m.confirmDialog.ErrorMode)
	require.Contains(t, m.confirmDialog.Message, "not a manager")
}

func TestKey_CtrlO_PromoteWorker(t *testing.T) {
	m := testModel()
	loadNodes(m, fakeNodes("worker"))
	m.Update(key("ctrl+o"))
	require.True(t, m.confirmDialog.Visible)
	require.Contains(t, m.confirmDialog.Message, "Promote")
}

func TestKey_CtrlO_AlreadyManager_ShowsError(t *testing.T) {
	m := testModel()
	entries := fakeNodes("mgr")
	entries[0].Manager = true
	loadNodes(m, entries)
	m.Update(key("ctrl+o"))
	require.True(t, m.confirmDialog.Visible)
	require.True(t, m.confirmDialog.ErrorMode)
	require.Contains(t, m.confirmDialog.Message, "already a manager")
}

func TestKey_A_OpensAvailabilityDialog(t *testing.T) {
	m := testModel()
	loadNodes(m, fakeNodes("mynode"))
	m.Update(key("a"))
	require.True(t, m.availabilityDialog)
}

func TestKey_CtrlL_OpensLabelInput(t *testing.T) {
	m := testModel()
	loadNodes(m, fakeNodes("mynode"))
	m.Update(key("ctrl+l"))
	require.True(t, m.labelInputDialog)
}

func TestKey_CtrlR_OpensLabelRemove(t *testing.T) {
	m := testModel()
	entries := fakeNodes("mynode")
	entries[0].Labels = map[string]string{"env": "prod"}
	loadNodes(m, entries)
	m.Update(key("ctrl+r"))
	require.True(t, m.labelRemoveDialog)
	require.Len(t, m.labelRemoveLabels, 1)
}

func TestKey_CtrlR_NoLabels_ShowsError(t *testing.T) {
	m := testModel()
	loadNodes(m, fakeNodes("mynode"))
	m.Update(key("ctrl+r"))
	require.True(t, m.confirmDialog.Visible)
	require.True(t, m.confirmDialog.ErrorMode)
	require.Contains(t, m.confirmDialog.Message, "no labels")
}

func TestKey_Q_Disabled(t *testing.T) {
	m := testModel()
	loadNodes(m, fakeNodes("n1"))
	m.Update(key("q"))
	require.True(t, m.Visible) // q is disabled, does nothing
}

// --- Sort key tests ---

func TestSortKey_H_Hostname(t *testing.T) {
	m := testModel()
	loadNodes(m, fakeNodes("b", "a"))
	m.Update(key("H"))
	require.Equal(t, SortByHostname, m.sortField)
	require.False(t, m.sortAscending) // toggles from default asc
}

func TestSortKey_S_State(t *testing.T) {
	m := testModel()
	loadNodes(m, fakeNodes("a", "b"))
	m.Update(key("S"))
	require.Equal(t, SortByState, m.sortField)
}

func TestSortKey_A_Availability(t *testing.T) {
	m := testModel()
	loadNodes(m, fakeNodes("a", "b"))
	m.Update(key("A"))
	require.Equal(t, SortByAvailability, m.sortField)
}

func TestSortKey_R_Role(t *testing.T) {
	m := testModel()
	loadNodes(m, fakeNodes("a", "b"))
	m.Update(key("R"))
	require.Equal(t, SortByRole, m.sortField)
}

func TestSortKey_V_Version(t *testing.T) {
	m := testModel()
	loadNodes(m, fakeNodes("a", "b"))
	m.Update(key("V"))
	require.Equal(t, SortByVersion, m.sortField)
}

func TestSortKey_D_Address(t *testing.T) {
	m := testModel()
	loadNodes(m, fakeNodes("a", "b"))
	m.Update(key("D"))
	require.Equal(t, SortByAddress, m.sortField)
}

func TestSortKey_L_Labels(t *testing.T) {
	m := testModel()
	loadNodes(m, fakeNodes("a", "b"))
	m.Update(key("L"))
	require.Equal(t, SortByLabels, m.sortField)
}

// --- Confirm dialog result tests ---

func TestConfirmResult_Demote(t *testing.T) {
	m := testModel()
	entries := fakeNodes("mgr")
	entries[0].Manager = true
	loadNodes(m, entries)
	m.confirmDialog.Visible = true
	m.confirmDialog.Message = "Demote node"
	cmd := m.Update(confirmdialog.ResultMsg{Confirmed: true})
	require.NotNil(t, cmd)
}

func TestConfirmResult_Promote(t *testing.T) {
	m := testModel()
	loadNodes(m, fakeNodes("worker"))
	m.confirmDialog.Visible = true
	m.confirmDialog.Message = "Promote node"
	cmd := m.Update(confirmdialog.ResultMsg{Confirmed: true})
	require.NotNil(t, cmd)
}

func TestConfirmResult_Remove(t *testing.T) {
	m := testModel()
	loadNodes(m, fakeNodes("target"))
	m.confirmDialog.Visible = true
	m.confirmDialog.Message = "Remove node"
	cmd := m.Update(confirmdialog.ResultMsg{Confirmed: true})
	require.NotNil(t, cmd)
}

func TestConfirmResult_Cancelled(t *testing.T) {
	m := testModel()
	m.confirmDialog.Visible = true
	m.Update(confirmdialog.ResultMsg{Confirmed: false})
	require.False(t, m.confirmDialog.Visible)
}

// --- Availability dialog key tests ---

func TestAvailabilityDialog_UpDown(t *testing.T) {
	m := testModel()
	m.availabilityDialog = true
	m.availabilitySelection = 0
	m.Update(key("down"))
	require.Equal(t, 1, m.availabilitySelection)
	m.Update(key("up"))
	require.Equal(t, 0, m.availabilitySelection)
}

func TestAvailabilityDialog_Esc(t *testing.T) {
	m := testModel()
	m.availabilityDialog = true
	m.Update(key("esc"))
	require.False(t, m.availabilityDialog)
}

func TestAvailabilityDialog_Enter(t *testing.T) {
	m := testModel()
	m.availabilityDialog = true
	m.availabilityNodeID = "node1"
	m.availabilitySelection = 0
	cmd := m.Update(key("enter"))
	require.False(t, m.availabilityDialog)
	require.NotNil(t, cmd)
}

// --- Label input dialog key tests ---

func TestLabelInputDialog_Esc(t *testing.T) {
	m := testModel()
	m.labelInputDialog = true
	m.labelInputValue = "test"
	m.Update(key("esc"))
	require.False(t, m.labelInputDialog)
	require.Empty(t, m.labelInputValue)
}

func TestLabelInputDialog_Enter_ValidLabel(t *testing.T) {
	m := testModel()
	m.labelInputDialog = true
	m.labelInputNodeID = "node1"
	m.labelInputValue = "env=prod"
	cmd := m.Update(key("enter"))
	require.False(t, m.labelInputDialog)
	require.NotNil(t, cmd)
}

func TestLabelInputDialog_Enter_InvalidFormat(t *testing.T) {
	m := testModel()
	m.labelInputDialog = true
	m.labelInputValue = "badformat"
	m.Update(key("enter"))
	require.False(t, m.labelInputDialog)
	require.True(t, m.confirmDialog.Visible)
	require.True(t, m.confirmDialog.ErrorMode)
	require.Contains(t, m.confirmDialog.Message, "key=value")
}

func TestLabelInputDialog_Backspace(t *testing.T) {
	m := testModel()
	m.labelInputDialog = true
	m.labelInputValue = "abc"
	m.Update(key("backspace"))
	require.Equal(t, "ab", m.labelInputValue)
}

// --- Label remove dialog key tests ---

func TestLabelRemoveDialog_Esc(t *testing.T) {
	m := testModel()
	m.labelRemoveDialog = true
	m.Update(key("esc"))
	require.False(t, m.labelRemoveDialog)
}

func TestLabelRemoveDialog_UpDown(t *testing.T) {
	m := testModel()
	m.labelRemoveDialog = true
	m.labelRemoveLabels = []string{"a=1", "b=2"}
	m.labelRemoveSelection = 0
	m.Update(key("down"))
	require.Equal(t, 1, m.labelRemoveSelection)
	m.Update(key("up"))
	require.Equal(t, 0, m.labelRemoveSelection)
}

func TestLabelRemoveDialog_Enter(t *testing.T) {
	m := testModel()
	m.labelRemoveDialog = true
	m.labelRemoveNodeID = "node1"
	m.labelRemoveLabels = []string{"env=prod"}
	m.labelRemoveSelection = 0
	cmd := m.Update(key("enter"))
	require.False(t, m.labelRemoveDialog)
	require.NotNil(t, cmd)
}

// --- LoadNodesCmd test ---

func TestDemoteNode_Timeout_ReturnsError(t *testing.T) {
	nodeMock := noopNodeOps()
	nodeMock.demoteNodeFn = func(_ context.Context, _ string) error {
		return context.DeadlineExceeded
	}
	m := testModel(func(m *Model) { m.deps.Nodes = nodeMock })
	entries := fakeNodes("mgr")
	entries[0].Manager = true
	loadNodes(m, entries)
	m.confirmDialog.Visible = true
	m.confirmDialog.Message = "Demote node mgr"
	cmd := m.Update(confirmdialog.ResultMsg{Confirmed: true})
	require.NotNil(t, cmd)
	msg := runCmd(cmd)
	_, ok := msg.(DemoteErrorMsg)
	require.True(t, ok, "expected DemoteErrorMsg, got %T", msg)
}

func TestLoadNodesCmd(t *testing.T) {
	snap := &docker.SwarmSnapshot{
		Nodes: []swarm.Node{
			{
				ID:          "n1",
				Description: swarm.NodeDescription{Hostname: "node1"},
				Spec:        swarm.NodeSpec{Availability: swarm.NodeAvailabilityActive},
				Status:      swarm.NodeStatus{State: swarm.NodeStateReady},
			},
		},
	}
	mockSnap := noopSnapshotOps()
	mockSnap.getSnapshotFn = func() *docker.SwarmSnapshot { return snap }
	m := testModel(func(m *Model) { m.deps.Snapshot = mockSnap })
	cmd := m.LoadNodesCmd()
	msg := runCmd(cmd)
	nodeMsg, ok := msg.(Msg)
	require.True(t, ok)
	require.Len(t, nodeMsg.Entries, 1)
	require.Equal(t, "node1", nodeMsg.Entries[0].Hostname)
}
