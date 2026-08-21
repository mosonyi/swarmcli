// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

//go:build integration

package network

import (
	"context"
	"fmt"
	"github.com/Eldara-Tech/swarmcli/v2/docker"
	swarmlog "github.com/Eldara-Tech/swarmcli/v2/utils/log"
	"testing"
	"time"

	"github.com/docker/docker/api/types/network"
	"github.com/stretchr/testify/require"
)

func uniqueName(base string) string {
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano())
}

func TestListNetworks(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()
	ctx := context.Background()

	networks, err := docker.ListNetworks(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, networks, "should have at least default networks")
}

func TestCreateAndRemoveNetwork(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()
	ctx := context.Background()

	name := uniqueName("test_net")
	id, _, err := docker.CreateNetwork(ctx, name, network.CreateOptions{
		Driver: "overlay",
	})
	require.NoError(t, err)
	t.Logf("Created network %s (ID=%s)", name, id)

	// Verify it exists
	info, err := docker.InspectNetwork(ctx, id)
	require.NoError(t, err)
	require.Equal(t, name, info.Name)

	// Remove it
	err = docker.RemoveNetwork(ctx, id)
	require.NoError(t, err)

	// Verify it's gone
	_, err = docker.InspectNetwork(ctx, id)
	require.Error(t, err)
}

func TestListServicesUsingNetwork(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()
	ctx := context.Background()

	// Create a network with no services attached
	name := uniqueName("test_net_svc")
	id, _, err := docker.CreateNetwork(ctx, name, network.CreateOptions{
		Driver: "overlay",
	})
	require.NoError(t, err)
	defer func() { _ = docker.RemoveNetwork(ctx, id) }()

	services, err := docker.ListServicesUsingNetwork(ctx, id, name)
	require.NoError(t, err)
	require.Empty(t, services, "new network should have no services")
}
