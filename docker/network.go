// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"
	"encoding/json"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// ListNetworks returns all networks in the swarm, on the ambient context.
func ListNetworks(ctx context.Context) ([]network.Summary, error) {
	cli, err := GetClient()
	if err != nil {
		return nil, err
	}
	return ListNetworksWith(ctx, cli)
}

// ListNetworksWith is ListNetworks against an explicit client.
func ListNetworksWith(ctx context.Context, cli *client.Client) ([]network.Summary, error) {
	return cli.NetworkList(ctx, network.ListOptions{})
}

// InspectNetwork returns detailed information about a network
func InspectNetwork(ctx context.Context, networkID string) (network.Inspect, error) {
	client, err := GetClient()
	if err != nil {
		return network.Inspect{}, err
	}
	return client.NetworkInspect(ctx, networkID, network.InspectOptions{})
}

// RemoveNetwork removes a network on the ambient context.
func RemoveNetwork(ctx context.Context, networkID string) error {
	cli, err := GetClient()
	if err != nil {
		return err
	}
	return RemoveNetworkWith(ctx, cli, networkID)
}

// RemoveNetworkWith is RemoveNetwork against an explicit client.
func RemoveNetworkWith(ctx context.Context, cli *client.Client, networkID string) error {
	return cli.NetworkRemove(ctx, networkID)
}

// CreateNetwork creates a new Docker network.
// Returns the created network ID and any daemon warnings.
func CreateNetwork(ctx context.Context, name string, opts network.CreateOptions) (string, []string, error) {
	cli, err := GetClient()
	if err != nil {
		return "", nil, err
	}
	return CreateNetworkWith(ctx, cli, name, opts)
}

// CreateNetworkWith is CreateNetwork against an explicit client.
func CreateNetworkWith(ctx context.Context, cli *client.Client, name string, opts network.CreateOptions) (string, []string, error) {
	resp, err := cli.NetworkCreate(ctx, name, opts)
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
