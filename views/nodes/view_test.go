// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package nodesview

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestView_NotVisible(t *testing.T) {
	m := testModel()
	m.Visible = false
	out := m.View()
	require.Empty(t, out)
}

func TestView_WithNodes(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.ready = true
	loadNodes(m, fakeNodes("node1", "node2"))
	m.List.Viewport.Width = 80
	m.List.Viewport.Height = 20
	out := m.View()
	require.Contains(t, out, "Nodes")
	require.Contains(t, out, "2 total")
}

func TestView_ShowsColumnHeaders(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.ready = true
	loadNodes(m, fakeNodes("n1"))
	m.List.Viewport.Width = 120
	m.List.Viewport.Height = 20
	out := m.View()
	require.Contains(t, out, "ID")
	require.Contains(t, out, "HOSTNAME")
}

func TestView_AvailabilityDialog(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.ready = true
	loadNodes(m, fakeNodes("n1"))
	m.List.Viewport.Width = 80
	m.List.Viewport.Height = 20
	m.availabilityDialog = true
	out := m.View()
	require.Contains(t, out, "Availability")
}

func TestView_LabelInputDialog(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.ready = true
	loadNodes(m, fakeNodes("n1"))
	m.List.Viewport.Width = 80
	m.List.Viewport.Height = 20
	m.labelInputDialog = true
	out := m.View()
	require.Contains(t, out, "Add Node Label")
}

func TestView_LabelRemoveDialog(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.ready = true
	loadNodes(m, fakeNodes("n1"))
	m.List.Viewport.Width = 80
	m.List.Viewport.Height = 20
	m.labelRemoveDialog = true
	m.labelRemoveLabels = []string{"env=prod"}
	out := m.View()
	require.Contains(t, out, "Remove Node Label")
}

func TestView_ManagerCount(t *testing.T) {
	m := testModel()
	m.Visible = true
	m.ready = true
	entries := fakeNodes("mgr", "worker")
	entries[0].Manager = true
	loadNodes(m, entries)
	m.List.Viewport.Width = 80
	m.List.Viewport.Height = 20
	out := m.View()
	require.Contains(t, out, "1 manager")
}
