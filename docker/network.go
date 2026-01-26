// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
)

// ListNetworks returns all networks in the swarm
func ListNetworks(ctx context.Context) ([]network.Summary, error) {
	client, err := GetClient()
	if err != nil {
		return nil, err
	}
	return client.NetworkList(ctx, network.ListOptions{})
}

// InspectNetwork returns detailed information about a network
func InspectNetwork(ctx context.Context, networkID string) (network.Inspect, error) {
	client, err := GetClient()
	if err != nil {
		return network.Inspect{}, err
	}
	return client.NetworkInspect(ctx, networkID, network.InspectOptions{})
}

// RemoveNetwork removes a network
func RemoveNetwork(ctx context.Context, networkID string) error {
	client, err := GetClient()
	if err != nil {
		return err
	}
	return client.NetworkRemove(ctx, networkID)
}

// CreateNetwork creates a new Docker network.
// Returns the created network ID and any daemon warnings.
func CreateNetwork(ctx context.Context, name string, opts network.CreateOptions) (string, []string, error) {
	client, err := GetClient()
	if err != nil {
		return "", nil, err
	}

	resp, err := client.NetworkCreate(ctx, name, opts)
	if err != nil {
		return "", nil, err
	}

	warnings := []string{}
	if resp.Warning != "" {
		warnings = append(warnings, resp.Warning)
	}

	return resp.ID, warnings, nil
}

// PruneNetworks removes all unused networks
func PruneNetworks(ctx context.Context) (network.PruneReport, error) {
	client, err := GetClient()
	if err != nil {
		return network.PruneReport{}, err
	}
	report, err := client.NetworksPrune(ctx, filters.Args{})
	return report, err
}

// ListServicesUsingNetwork returns all services that are connected to a network.
// In Swarm, service network targets can be specified by ID or by name.
func ListServicesUsingNetwork(ctx context.Context, networkID, networkName string) ([]string, error) {
	client, err := GetClient()
	if err != nil {
		return nil, err
	}

	services, err := client.ServiceList(ctx, swarm.ServiceListOptions{})
	if err != nil {
		return nil, err
	}

	var connectedServices []string
	for _, svc := range services {
		for _, net := range svc.Spec.TaskTemplate.Networks {
			if net.Target == networkID || (networkName != "" && net.Target == networkName) {
				connectedServices = append(connectedServices, svc.Spec.Name)
				break
			}
		}
	}
	return connectedServices, nil
}

// NetworkWithUsage is a helper struct that includes usage information
type NetworkWithUsage struct {
	Network  network.Summary
	Services []string // Services using this network
}

func (nw *NetworkWithUsage) JSON() ([]byte, error) {
	type jsonNetwork struct {
		Network  network.Summary `json:"Network"`
		Services []string        `json:"Services,omitempty"`
	}

	obj := jsonNetwork{
		Network:  nw.Network,
		Services: nw.Services,
	}

	return json.Marshal(obj)
}

// PrettyJSON returns the JSON representation of the network,
// but pretty-printed (indented) for human-readable viewing.
func (nw *NetworkWithUsage) PrettyJSON() ([]byte, error) {
	raw, err := nw.JSON()
	if err != nil {
		return nil, err
	}

	var buf []byte
	buf, err = json.MarshalIndent(json.RawMessage(raw), "", "  ")
	if err != nil {
		return nil, fmt.Errorf("pretty-print failed: %w", err)
	}
	return buf, nil
}
