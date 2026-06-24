// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/swarm"
)

// nodeUpdateMaxAttempts bounds how many times a spec update is retried when the
// daemon rejects it with "update out of sequence". Swarm bumps a node's version
// index on background status/heartbeat writes as well as spec changes, so a
// fetch-then-update can race a concurrent bump and submit a stale version index;
// re-fetching and retrying resolves it.
const nodeUpdateMaxAttempts = 5

// nodeUpdater is the subset of the Docker client used to mutate a node's spec.
// *client.Client satisfies it; it exists so updateNodeSpec is unit-testable
// without a live daemon.
type nodeUpdater interface {
	NodeInspectWithRaw(ctx context.Context, nodeID string) (swarm.Node, []byte, error)
	NodeUpdate(ctx context.Context, nodeID string, version swarm.Version, spec swarm.NodeSpec) error
}

// isUpdateOutOfSequence reports whether err is the daemon's optimistic-concurrency
// rejection ("update out of sequence"), raised when NodeUpdate is given a version
// index that no longer matches the store.
func isUpdateOutOfSequence(err error) bool {
	return err != nil && strings.Contains(strings.ToLower(err.Error()), "update out of sequence")
}

// updateNodeSpec fetches the node, applies mutate to a copy of its spec, and
// submits the update using the node's current version index. On an "update out
// of sequence" rejection it re-fetches and retries, up to nodeUpdateMaxAttempts.
func updateNodeSpec(ctx context.Context, c nodeUpdater, nodeID string, mutate func(*swarm.NodeSpec)) error {
	var lastErr error
	for attempt := 0; attempt < nodeUpdateMaxAttempts; attempt++ {
		node, _, err := c.NodeInspectWithRaw(ctx, nodeID)
		if err != nil {
			return fmt.Errorf("inspect node: %w", err)
		}

		spec := node.Spec
		mutate(&spec)

		err = c.NodeUpdate(ctx, nodeID, node.Version, spec)
		if err == nil {
			return nil
		}
		if !isUpdateOutOfSequence(err) {
			return err
		}
		lastErr = err
	}
	return fmt.Errorf("after %d attempts: %w", nodeUpdateMaxAttempts, lastErr)
}

// GetLocalNodeID returns the swarm node ID of the daemon the active Docker
// client is connected to, or "" if it is not an active swarm node.
func GetLocalNodeID(ctx context.Context) (string, error) {
	c, err := GetClient()
	if err != nil {
		return "", err
	}

	info, err := c.Info(ctx)
	if err != nil {
		return "", fmt.Errorf("query docker info: %w", err)
	}
	return info.Swarm.NodeID, nil
}

func GetNodeIDToHostnameMapFromDocker(ctx context.Context) (map[string]string, error) {
	c, err := GetClient()
	if err != nil {
		return nil, err
	}

	nodes, err := c.NodeList(ctx, swarm.NodeListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing nodes: %w", err)
	}

	m := make(map[string]string, len(nodes))
	for _, n := range nodes {
		m[n.ID] = n.Description.Hostname
	}
	return m, nil
}

// DemoteNode sets the node role to worker (demotes a manager).
func DemoteNode(ctx context.Context, nodeID string) error {
	c, err := GetClient()
	if err != nil {
		return err
	}

	if err := updateNodeSpec(ctx, c, nodeID, func(spec *swarm.NodeSpec) {
		spec.Role = swarm.NodeRoleWorker
	}); err != nil {
		return fmt.Errorf("demote node: %w", err)
	}
	return nil
}

// PromoteNode sets the node role to manager (promotes a worker).
func PromoteNode(ctx context.Context, nodeID string) error {
	c, err := GetClient()
	if err != nil {
		return err
	}

	if err := updateNodeSpec(ctx, c, nodeID, func(spec *swarm.NodeSpec) {
		spec.Role = swarm.NodeRoleManager
	}); err != nil {
		return fmt.Errorf("promote node: %w", err)
	}
	return nil
}

// SetNodeAvailability sets the availability of a node (active, pause, drain).
func SetNodeAvailability(ctx context.Context, nodeID string, availability swarm.NodeAvailability) error {
	c, err := GetClient()
	if err != nil {
		return err
	}

	if err := updateNodeSpec(ctx, c, nodeID, func(spec *swarm.NodeSpec) {
		spec.Availability = availability
	}); err != nil {
		return fmt.Errorf("set node availability: %w", err)
	}
	return nil
}

// AddNodeLabel adds or updates a label on a node.
func AddNodeLabel(ctx context.Context, nodeID string, key string, value string) error {
	c, err := GetClient()
	if err != nil {
		return err
	}

	if err := updateNodeSpec(ctx, c, nodeID, func(spec *swarm.NodeSpec) {
		if spec.Labels == nil {
			spec.Labels = make(map[string]string)
		}
		spec.Labels[key] = value
	}); err != nil {
		return fmt.Errorf("add node label: %w", err)
	}
	return nil
}

// RemoveNodeLabel removes a label from a node
func RemoveNodeLabel(ctx context.Context, nodeID string, key string) error {
	c, err := GetClient()
	if err != nil {
		return err
	}

	if err := updateNodeSpec(ctx, c, nodeID, func(spec *swarm.NodeSpec) {
		if spec.Labels != nil {
			delete(spec.Labels, key)
		}
	}); err != nil {
		return fmt.Errorf("remove node label: %w", err)
	}
	return nil
}

// RemoveNode removes a node from the swarm.
func RemoveNode(ctx context.Context, nodeID string, force bool) error {
	c, err := GetClient()
	if err != nil {
		return err
	}

	opts := swarm.NodeRemoveOptions{Force: force}
	if err := c.NodeRemove(ctx, nodeID, opts); err != nil {
		return fmt.Errorf("remove node: %w", err)
	}
	return nil
}
