package docker

import (
	"context"
	"fmt"
	"sort"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

type StackService struct {
	NodeID      string
	StackName   string
	ServiceName string
}

// GetServicesInStack returns the services belonging to a given stack.
func GetServicesInStack(stackName string) []StackService {
	c, err := GetClient()
	if err != nil {
		fmt.Println("failed to init docker client:", err)
		return nil
	}
	defer c.Close()

	ctx := context.Background()

	f := filters.NewArgs()
	f.Add("label", fmt.Sprintf("com.docker.stack.namespace=%s", stackName))

	services, err := c.ServiceList(ctx, types.ServiceListOptions{Filters: f})
	if err != nil {
		fmt.Println("failed to list services for stack:", stackName, ":", err)
		return nil
	}

	var stackServices []StackService
	for _, svc := range services {
		stackServices = append(stackServices, StackService{
			StackName:   stackName,
			ServiceName: svc.Spec.Name,
		})
	}

	sort.Slice(stackServices, func(i, j int) bool {
		return stackServices[i].ServiceName < stackServices[j].ServiceName
	})
	return stackServices
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
