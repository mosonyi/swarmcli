// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"testing"

	"github.com/docker/docker/api/types/volume"
	"github.com/stretchr/testify/require"
)

func TestVolumeInfoFromSummary_FullMapping(t *testing.T) {
	v := &volume.Volume{
		Name:       "vol1",
		Driver:     "local",
		Mountpoint: "/var/lib/docker/volumes/vol1/_data",
		CreatedAt:  "2026-01-02T15:04:05Z",
		Labels:     map[string]string{"com.docker.stack.namespace": "mystack"},
	}
	info := volumeInfoFromSummary(v, "node-1")

	require.Equal(t, "vol1", info.Name)
	require.Equal(t, "mystack", info.Stack)
	require.Equal(t, "local", info.Driver)
	require.Equal(t, "/var/lib/docker/volumes/vol1/_data", info.Mountpoint)
	require.Equal(t, "node-1", info.Host)
	require.Equal(t, 2026, info.Created.UTC().Year())
	require.Same(t, v, info.Raw)
}

func TestVolumeInfoFromSummary_EmptyOptionalFields(t *testing.T) {
	info := volumeInfoFromSummary(&volume.Volume{Name: "anon", Driver: "local"}, "")
	require.Empty(t, info.Stack)
	require.Empty(t, info.Host)
	require.True(t, info.Created.IsZero())
}

func TestVolumeInfoFromSummary_InvalidTimestamp(t *testing.T) {
	info := volumeInfoFromSummary(&volume.Volume{Name: "x", CreatedAt: "not-a-time"}, "n")
	require.True(t, info.Created.IsZero())
}
