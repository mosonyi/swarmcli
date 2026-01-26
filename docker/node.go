// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/swarm"
)

func GetNodeIDToHostnameMapFromDocker() (map[string]string, error) {
	c, err := GetClient()
	if err != nil {
		return nil, err
	}
	defer closeCli(c)

	nodes, err := c.NodeList(context.Background(), swarm.NodeListOptions{})
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
	defer closeCli(c)

	// Fetch current node to get the version and current spec
	node, _, err := c.NodeInspectWithRaw(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("inspect node: %w", err)
	}

	// Modify spec to set worker role
	spec := node.Spec
	spec.Role = swarm.NodeRoleWorker

	// Perform update using the node's current version index
	if err := c.NodeUpdate(ctx, nodeID, node.Version, spec); err != nil {
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
	defer closeCli(c)

	// Fetch current node to get the version and current spec
	node, _, err := c.NodeInspectWithRaw(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("inspect node: %w", err)
	}

	// Modify spec to set manager role
	spec := node.Spec
	spec.Role = swarm.NodeRoleManager

	// Perform update using the node's current version index
	if err := c.NodeUpdate(ctx, nodeID, node.Version, spec); err != nil {
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
	defer closeCli(c)

	// Fetch current node to get the version and current spec
	node, _, err := c.NodeInspectWithRaw(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("inspect node: %w", err)
	}

	// Modify spec to set availability
	spec := node.Spec
	spec.Availability = availability

	// Perform update using the node's current version index
	if err := c.NodeUpdate(ctx, nodeID, node.Version, spec); err != nil {
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
	defer closeCli(c)

	// Fetch current node to get the version and current spec
	node, _, err := c.NodeInspectWithRaw(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("inspect node: %w", err)
	}

	// Modify spec to add/update label
	spec := node.Spec
	if spec.Labels == nil {
		spec.Labels = make(map[string]string)
	}
	spec.Labels[key] = value

	// Perform update using the node's current version index
	if err := c.NodeUpdate(ctx, nodeID, node.Version, spec); err != nil {
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
	defer closeCli(c)

	// Fetch current node to get the version and current spec
	node, _, err := c.NodeInspectWithRaw(ctx, nodeID)
	if err != nil {
		return fmt.Errorf("inspect node: %w", err)
	}

	// Modify spec to remove label
	spec := node.Spec
	if spec.Labels != nil {
		delete(spec.Labels, key)
	}

	// Perform update using the node's current version index
	if err := c.NodeUpdate(ctx, nodeID, node.Version, spec); err != nil {
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
	defer closeCli(c)

	opts := swarm.NodeRemoveOptions{Force: force}
	if err := c.NodeRemove(ctx, nodeID, opts); err != nil {
		return fmt.Errorf("remove node: %w", err)
	}
	return nil
}
