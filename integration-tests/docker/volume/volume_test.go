// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

//go:build integration

package volume

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/Eldara-Tech/swarmcli/v2/docker"
	swarmlog "github.com/Eldara-Tech/swarmcli/v2/utils/log"

	"github.com/docker/docker/api/types/volume"
	"github.com/stretchr/testify/require"
)

func uniqueName(base string) string {
	return fmt.Sprintf("%s-%d", base, time.Now().UnixNano())
}

// TestListVolumes_IncludesCreatedVolume verifies the connected-node listing
// path end-to-end. docker volume ls is per-node, so rather than depend on
// where swarm scheduled the test stack's tasks, we create a volume on the
// connected (manager) node and assert it surfaces with mapped fields.
func TestListVolumes_IncludesCreatedVolume(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()
	ctx := context.Background()

	cli, err := docker.GetClient()
	require.NoError(t, err)

	name := uniqueName("test_vol")
	_, err = cli.VolumeCreate(ctx, volume.CreateOptions{
		Name:   name,
		Driver: "local",
		Labels: map[string]string{"com.docker.stack.namespace": "demo"},
	})
	require.NoError(t, err)
	t.Cleanup(func() { _ = cli.VolumeRemove(ctx, name, true) })

	volumes, err := docker.ListVolumes(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, volumes)

	var found *docker.VolumeInfo
	for i := range volumes {
		if volumes[i].Name == name {
			found = &volumes[i]
			break
		}
	}
	require.NotNil(t, found, "created volume %q should appear in ListVolumes", name)
	require.Equal(t, "local", found.Driver)
	require.Equal(t, "demo", found.Stack)
	require.NotEmpty(t, found.Mountpoint)
	require.NotEmpty(t, found.Host, "host should be the connected node's hostname")
}

// TestDefaultVolumeOps_ListsViaInterface exercises the VolumeOps seam BE swaps:
// the default (connected-node) implementation lists the volume and leaves
// NodeID empty (the field an aggregating implementation fills).
func TestDefaultVolumeOps_ListsViaInterface(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()
	ctx := context.Background()

	cli, err := docker.GetClient()
	require.NoError(t, err)

	name := uniqueName("test_vol_ops")
	_, err = cli.VolumeCreate(ctx, volume.CreateOptions{Name: name, Driver: "local"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = cli.VolumeRemove(ctx, name, true) })

	ops := docker.DefaultDeps().Volumes
	volumes, err := ops.ListVolumes(ctx)
	require.NoError(t, err)

	var found *docker.VolumeInfo
	for i := range volumes {
		if volumes[i].Name == name {
			found = &volumes[i]
			break
		}
	}
	require.NotNil(t, found, "volume %q should appear via the VolumeOps seam", name)
	require.Empty(t, found.NodeID, "CE single-node impl leaves NodeID empty")
}

func TestInspectVolume(t *testing.T) {
	swarmlog.InitTestIfTestLogEnv()
	ctx := context.Background()

	cli, err := docker.GetClient()
	require.NoError(t, err)

	name := uniqueName("test_vol_inspect")
	_, err = cli.VolumeCreate(ctx, volume.CreateOptions{Name: name, Driver: "local"})
	require.NoError(t, err)
	t.Cleanup(func() { _ = cli.VolumeRemove(ctx, name, true) })

	v, err := docker.InspectVolume(ctx, name)
	require.NoError(t, err)
	require.Equal(t, name, v.Name)
	require.Equal(t, "local", v.Driver)
}
