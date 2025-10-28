package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
)

type StackService struct {
	NodeID      string
	StackName   string
	ServiceName string
}

func GetServiceNameToIDMap() (map[string]string, error) {
	c, err := GetClient()
	if err != nil {
		return nil, err
	}
	defer c.Close()

	services, err := c.ServiceList(context.Background(), types.ServiceListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}

	m := make(map[string]string, len(services))
	for _, s := range services {
		m[s.Spec.Name] = s.ID
	}
	return m, nil
}
