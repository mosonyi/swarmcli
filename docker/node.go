package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
)

func GetNodeIDs() ([]string, error) {
	c, err := GetClient()
	if err != nil {
		return nil, err
	}

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

func GetNodeTaskNames(nodeID string) ([]string, error) {
	c, err := GetClient()
	if err != nil {
		return nil, err
	}

	tasks, err := c.TaskList(context.Background(), types.TaskListOptions{})
	if err != nil {
		return nil, err
	}

	var taskNames []string
	for _, t := range tasks {
		if t.NodeID == nodeID && t.Status.State == "running" {
			taskNames = append(taskNames, t.ServiceID)
		}
	}
	return taskNames, nil
}
