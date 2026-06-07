// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"
	"time"

	"github.com/docker/docker/api/types/volume"
)

// VolumeInfo is the edition-agnostic view of a Docker volume consumed by the
// volumes view. The default (CE) implementation populates it from the
// connected node only; the Host field is an extension point so an
// implementation that aggregates across all swarm nodes can report which node
// each volume lives on.
type VolumeInfo struct {
	Name       string
	Stack      string // com.docker.stack.namespace label, "" if not stack-managed
	Driver     string
	Mountpoint string
	Created    time.Time
	Host       string // node the volume lives on
	Labels     map[string]string
	Raw        *volume.Volume // underlying SDK object, for inspect
}

// ListVolumes returns the volumes on the connected Docker node.
//
// docker volume ls is per-node: this lists only the volumes local to the
// daemon the current context points at. Listing volumes across every swarm
// node requires reaching each node individually and is left as an extension
// point (see VolumeOps).
func ListVolumes(ctx context.Context) ([]VolumeInfo, error) {
	cli, err := GetClient()
	if err != nil {
		return nil, err
	}

	resp, err := cli.VolumeList(ctx, volume.ListOptions{})
	if err != nil {
		return nil, err
	}

	// The connected daemon's hostname is the node every listed volume lives on.
	host := ""
	if info, infoErr := cli.Info(ctx); infoErr == nil {
		host = info.Name
	}

	items := make([]VolumeInfo, 0, len(resp.Volumes))
	for _, v := range resp.Volumes {
		if v == nil {
			continue
		}
		items = append(items, volumeInfoFromSummary(v, host))
	}
	return items, nil
}

// InspectVolume returns the raw SDK volume for the given name on the
// connected node.
func InspectVolume(ctx context.Context, name string) (volume.Volume, error) {
	cli, err := GetClient()
	if err != nil {
		return volume.Volume{}, err
	}
	return cli.VolumeInspect(ctx, name)
}

// volumeInfoFromSummary maps an SDK volume to a VolumeInfo, parsing the
// creation timestamp and extracting the stack label. host is the node the
// volume lives on.
func volumeInfoFromSummary(v *volume.Volume, host string) VolumeInfo {
	var created time.Time
	if v.CreatedAt != "" {
		if t, err := time.Parse(time.RFC3339, v.CreatedAt); err == nil {
			created = t
		}
	}
	return VolumeInfo{
		Name:       v.Name,
		Stack:      v.Labels["com.docker.stack.namespace"],
		Driver:     v.Driver,
		Mountpoint: v.Mountpoint,
		Created:    created,
		Host:       host,
		Labels:     v.Labels,
		Raw:        v,
	}
}
