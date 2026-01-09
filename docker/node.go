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
