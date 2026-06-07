// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package volumesview

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"swarmcli/core/primitives/hash"
	"swarmcli/docker"
	inspectview "swarmcli/views/inspect"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

const PollInterval = 5 * time.Second

type TickMsg time.Time

// PollRetryMsg signals that polling found no changes; the Update handler
// should schedule the next tick.
type PollRetryMsg struct{}

type SpinnerTickMsg time.Time

type VolumesLoadedMsg struct {
	Volumes []volumeItem
	Err     error
	// Warn is a non-fatal banner message. Set when ListVolumes reports a
	// docker.PartialListError: the data is usable but some nodes were
	// unreachable. Empty on a fully successful load (which clears the banner).
	Warn string
}

func (m *Model) loadVolumesCmd() tea.Cmd {
	volumeOps := m.deps.Volumes
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		volumes, err := volumeOps.ListVolumes(ctx)
		if err != nil {
			var partial *docker.PartialListError
			if errors.As(err, &partial) {
				return VolumesLoadedMsg{Volumes: toVolumeItems(volumes), Warn: partial.Error()}
			}
			return VolumesLoadedMsg{Err: fmt.Errorf("failed to list volumes: %w", err)}
		}
		return VolumesLoadedMsg{Volumes: toVolumeItems(volumes)}
	}
}

// checkVolumesCmd reloads only when the volume set has changed since lastHash.
func (m *Model) checkVolumesCmd(lastHash uint64) tea.Cmd {
	volumeOps := m.deps.Volumes
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		volumes, err := volumeOps.ListVolumes(ctx)
		warn := ""
		if err != nil {
			var partial *docker.PartialListError
			if !errors.As(err, &partial) {
				return VolumesLoadedMsg{Err: err}
			}
			// Partial failure: the returned volumes are still usable.
			warn = partial.Error()
		}
		items := toVolumeItems(volumes)
		newHash, hErr := hash.Compute(stableVolumes(items))
		if hErr != nil {
			l().Errorf("Error computing hash: %v", hErr)
			return PollRetryMsg{}
		}
		if newHash != lastHash {
			l().Info("Volumes changed, reloading")
			return VolumesLoadedMsg{Volumes: items, Warn: warn}
		}
		return PollRetryMsg{}
	}
}

func (m *Model) inspectVolumeCmd(name string) tea.Cmd {
	volumeOps := m.deps.Volumes
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		title := fmt.Sprintf("Volume: %s", name)

		vol, err := volumeOps.InspectVolume(ctx, name)
		if err != nil {
			return view.NavigateToMsg{
				ViewName: inspectview.ViewName,
				Payload: map[string]any{
					"title":  title,
					"json":   fmt.Sprintf("# Error inspecting volume:\n# %v", err),
					"format": inspectview.FormatRaw,
				},
			}
		}

		content, jsonErr := json.MarshalIndent(vol, "", "  ")
		if jsonErr != nil {
			return view.NavigateToMsg{
				ViewName: inspectview.ViewName,
				Payload: map[string]any{
					"title":  title,
					"json":   fmt.Sprintf("# Error marshalling volume:\n# %v", jsonErr),
					"format": inspectview.FormatRaw,
				},
			}
		}

		return view.NavigateToMsg{
			ViewName: inspectview.ViewName,
			Payload: map[string]any{
				"title": title,
				"json":  string(content),
			},
		}
	}
}

func toVolumeItems(volumes []docker.VolumeInfo) []volumeItem {
	items := make([]volumeItem, len(volumes))
	for i, v := range volumes {
		items[i] = volumeItem{
			Name:       v.Name,
			Stack:      v.Stack,
			Driver:     v.Driver,
			Mountpoint: v.Mountpoint,
			Created:    v.Created,
			Host:       v.Host,
			NodeID:     v.NodeID,
		}
	}
	return items
}

// stableVolumes projects the fields that define a meaningful change for
// change-detection hashing.
func stableVolumes(items []volumeItem) []struct {
	Name, Driver, Mountpoint, Host string
} {
	out := make([]struct {
		Name, Driver, Mountpoint, Host string
	}, len(items))
	for i, it := range items {
		out[i] = struct {
			Name, Driver, Mountpoint, Host string
		}{it.Name, it.Driver, it.Mountpoint, it.Host}
	}
	return out
}
