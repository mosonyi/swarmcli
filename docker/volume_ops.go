// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"

	"github.com/docker/docker/api/types/volume"
)

// VolumeOps abstracts volume operations for testability and extensibility.
//
// The default implementation lists volumes on the connected node only.
// Implementations that aggregate volumes across all swarm nodes can be
// substituted via Deps without changing the volumes view.
type VolumeOps interface {
	ListVolumes(ctx context.Context) ([]VolumeInfo, error)
	InspectVolume(ctx context.Context, name string) (volume.Volume, error)
}

type defaultVolumeOps struct{}

func (defaultVolumeOps) ListVolumes(ctx context.Context) ([]VolumeInfo, error) {
	return ListVolumes(ctx)
}
func (defaultVolumeOps) InspectVolume(ctx context.Context, name string) (volume.Volume, error) {
	return InspectVolume(ctx, name)
}
