package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
)

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
