// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
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
	Host       string // node hostname the volume lives on (display)
	NodeID     string // swarm node ID the volume lives on; "" for the CE single-node impl, filled by aggregating implementations for node-addressed actions
	Labels     map[string]string
	Raw        *volume.Volume // underlying SDK object, for inspect
}

// PartialListError reports that a listing succeeded but is degraded: the
// returned items are valid and shown, with a non-fatal banner explaining the
// limitation, instead of failing outright. An aggregating implementation
// returns it when some nodes are unreachable (NodeErrors), or with a custom
// Note when the listing fell back to a narrower scope (e.g. connected-node
// only because the cross-node path is unavailable). The default single-node
// implementation never returns it.
type PartialListError struct {
	NodeErrors map[string]string // node identifier -> error summary
	Note       string            // optional banner override; takes precedence over NodeErrors
}

func (e *PartialListError) Error() string {
	if e.Note != "" {
		return e.Note
	}
	if len(e.NodeErrors) == 1 {
		return "1 node unreachable"
	}
	return fmt.Sprintf("%d nodes unreachable", len(e.NodeErrors))
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
	return ListVolumesWith(ctx, cli)
}

// ListVolumesWith is ListVolumes against an explicit client.
func ListVolumesWith(ctx context.Context, cli *client.Client) ([]VolumeInfo, error) {
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

// RemoveVolume removes a named volume on the connected node. force deletes it
// even if it is referenced. Like ListVolumes, this acts on the connected node
// only; cross-node removal is left as an extension point.
func RemoveVolume(ctx context.Context, name string, force bool) error {
	c, err := GetClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	return RemoveVolumeWith(ctx, c, name, force)
}

// RemoveVolumeWith is RemoveVolume against an explicit client.
func RemoveVolumeWith(ctx context.Context, cli *client.Client, name string, force bool) error {
	if err := cli.VolumeRemove(ctx, name, force); err != nil {
		return fmt.Errorf("removing volume %q: %w", name, err)
	}
	return nil
}
