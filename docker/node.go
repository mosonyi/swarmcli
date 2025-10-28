package docker

import (
	"context"
	"fmt"
	"log"

	"github.com/docker/docker/api/types"
)

// ListSwarmNodes returns all swarm nodes.
func ListSwarmNodes() ([]SwarmNode, error) {
	c, err := GetClient()

	if err != nil {
		return nil, err
	}

	nodes, err := c.NodeList(context.Background(), types.NodeListOptions{})
	if err != nil {
		log.Println("NodeList error:", err)
		return nil, err
	}

	res := make([]SwarmNode, 0, len(nodes))
	for _, n := range nodes {
		managerStatus := ""
		if n.ManagerStatus != nil {
			managerStatus = string(n.ManagerStatus.Reachability)
		}
		res = append(res, SwarmNode{
			ID:            n.ID,
			Hostname:      n.Description.Hostname,
			Status:        string(n.Status.State),
			Availability:  string(n.Spec.Availability),
			ManagerStatus: managerStatus,
		})
	}
	return res, nil
}

func GetNodeIDs() ([]string, error) {
	c, err := GetClient()
	if err != nil {
		return nil, err
	}
	defer c.Close()

	nodes, err := c.NodeList(context.Background(), types.NodeListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing nodes: %w", err)
	}

	ids := make([]string, len(nodes))
	for i, n := range nodes {
		ids[i] = n.ID
	}
	return ids, nil
}

func GetNodeIDToHostnameMapFromDocker() (map[string]string, error) {
	c, err := GetClient()
	if err != nil {
		return nil, err
	}
	defer c.Close()

	nodes, err := c.NodeList(context.Background(), types.NodeListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing nodes: %w", err)
	}

	m := make(map[string]string, len(nodes))
	for _, n := range nodes {
		m[n.ID] = n.Description.Hostname
	}
	return m, nil
}
