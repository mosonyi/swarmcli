// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"

	"github.com/docker/docker/api/types/network"
)

// NetworkOps abstracts network operations for testability and extensibility.
type NetworkOps interface {
	ListNetworks(ctx context.Context) ([]network.Summary, error)
	InspectNetwork(ctx context.Context, networkID string) (network.Inspect, error)
	RemoveNetwork(ctx context.Context, networkID string) error
	CreateNetwork(ctx context.Context, name string, opts network.CreateOptions) (string, []string, error)
	PruneNetworks(ctx context.Context) (network.PruneReport, error)
	ListServicesUsingNetwork(ctx context.Context, networkID, networkName string) ([]string, error)
}

type defaultNetworkOps struct{}

func (defaultNetworkOps) ListNetworks(ctx context.Context) ([]network.Summary, error) {
	return ListNetworks(ctx)
}
func (defaultNetworkOps) InspectNetwork(ctx context.Context, networkID string) (network.Inspect, error) {
	return InspectNetwork(ctx, networkID)
}
func (defaultNetworkOps) RemoveNetwork(ctx context.Context, networkID string) error {
	return RemoveNetwork(ctx, networkID)
}
func (defaultNetworkOps) CreateNetwork(ctx context.Context, name string, opts network.CreateOptions) (string, []string, error) {
	return CreateNetwork(ctx, name, opts)
}
func (defaultNetworkOps) PruneNetworks(ctx context.Context) (network.PruneReport, error) {
	return PruneNetworks(ctx)
}
func (defaultNetworkOps) ListServicesUsingNetwork(ctx context.Context, networkID, networkName string) ([]string, error) {
	return ListServicesUsingNetwork(ctx, networkID, networkName)
}
