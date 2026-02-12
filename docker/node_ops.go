// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"

	"github.com/docker/docker/api/types/swarm"
)

// NodeOps abstracts node operations for testability and extensibility.
type NodeOps interface {
	GetNodeIDToHostnameMapFromDocker() (map[string]string, error)
	DemoteNode(ctx context.Context, nodeID string) error
	PromoteNode(ctx context.Context, nodeID string) error
	SetNodeAvailability(ctx context.Context, nodeID string, availability swarm.NodeAvailability) error
	AddNodeLabel(ctx context.Context, nodeID, key, value string) error
	RemoveNodeLabel(ctx context.Context, nodeID, key string) error
	RemoveNode(ctx context.Context, nodeID string, force bool) error
}

type defaultNodeOps struct{}

func (defaultNodeOps) GetNodeIDToHostnameMapFromDocker() (map[string]string, error) {
	return GetNodeIDToHostnameMapFromDocker()
}
func (defaultNodeOps) DemoteNode(ctx context.Context, nodeID string) error {
	return DemoteNode(ctx, nodeID)
}
func (defaultNodeOps) PromoteNode(ctx context.Context, nodeID string) error {
	return PromoteNode(ctx, nodeID)
}
func (defaultNodeOps) SetNodeAvailability(ctx context.Context, nodeID string, availability swarm.NodeAvailability) error {
	return SetNodeAvailability(ctx, nodeID, availability)
}
func (defaultNodeOps) AddNodeLabel(ctx context.Context, nodeID, key, value string) error {
	return AddNodeLabel(ctx, nodeID, key, value)
}
func (defaultNodeOps) RemoveNodeLabel(ctx context.Context, nodeID, key string) error {
	return RemoveNodeLabel(ctx, nodeID, key)
}
func (defaultNodeOps) RemoveNode(ctx context.Context, nodeID string, force bool) error {
	return RemoveNode(ctx, nodeID, force)
}
