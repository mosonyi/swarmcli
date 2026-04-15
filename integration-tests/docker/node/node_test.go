// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

//go:build integration

package node

import (
	"context"
	"swarmcli/docker"
	swarmlog "swarmcli/utils/log"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestListNodes(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()

	snap, err := docker.RefreshSnapshot()
	require.NoError(t, err)
	require.NotEmpty(t, snap.Nodes, "swarm should have at least one node")
}

func TestNodeEntries(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()

	snap, err := docker.RefreshSnapshot()
	require.NoError(t, err)

	entries := snap.ToNodeEntries()
	require.NotEmpty(t, entries)

	for _, e := range entries {
		require.NotEmpty(t, e.Hostname, "each node should have a hostname")
		require.NotEmpty(t, e.Role, "each node should have a role")
		require.NotEmpty(t, e.State, "each node should have a state")
		require.NotEmpty(t, e.Availability, "each node should have availability")
		t.Logf("Node: %s role=%s state=%s avail=%s manager=%v",
			e.Hostname, e.Role, e.State, e.Availability, e.Manager)
	}
}

func TestAddAndRemoveNodeLabel(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()
	ctx := context.Background()

	snap, err := docker.RefreshSnapshot()
	require.NoError(t, err)
	require.NotEmpty(t, snap.Nodes)

	nodeID := snap.Nodes[0].ID
	labelKey := "swarmcli-test-label"
	labelValue := "test-value"

	// Add label
	err = docker.AddNodeLabel(ctx, nodeID, labelKey, labelValue)
	require.NoError(t, err, "AddNodeLabel should succeed")
	t.Logf("Added label %s=%s to node %s", labelKey, labelValue, nodeID)

	// Verify label exists
	snap2, err := docker.RefreshSnapshot()
	require.NoError(t, err)
	entries := snap2.ToNodeEntries()
	var found bool
	for _, e := range entries {
		if e.ID == nodeID {
			val, ok := e.Labels[labelKey]
			require.True(t, ok, "label should exist on node")
			require.Equal(t, labelValue, val)
			found = true
			break
		}
	}
	require.True(t, found, "node should be found in entries")

	// Remove label
	err = docker.RemoveNodeLabel(ctx, nodeID, labelKey)
	require.NoError(t, err, "RemoveNodeLabel should succeed")

	// Verify label is gone
	snap3, err := docker.RefreshSnapshot()
	require.NoError(t, err)
	for _, n := range snap3.Nodes {
		if n.ID == nodeID {
			_, ok := n.Spec.Labels[labelKey]
			require.False(t, ok, "label should be removed")
			break
		}
	}
}
