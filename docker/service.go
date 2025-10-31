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

// GetServicesInStackOnNode returns the services belonging to a given stack *and node*.
func GetServicesInStackOnNode(stackName, nodeID string) []StackService {
	c, err := GetClient()
	if err != nil {
		fmt.Println("failed to init docker client:", err)
		return nil
	}
	defer c.Close()

	ctx := context.Background()

	// 1. List all tasks on this node
	tasks, err := c.TaskList(ctx, types.TaskListOptions{
		Filters: filters.NewArgs(filters.Arg("node", nodeID)),
	})
	if err != nil {
		fmt.Println("failed to list tasks for node:", nodeID, ":", err)
		return nil
	}

	// 2. Gather service IDs that belong to this node
	nodeServiceIDs := make(map[string]struct{})
	for _, t := range tasks {
		if t.ServiceID != "" {
			nodeServiceIDs[t.ServiceID] = struct{}{}
		}
	}

	// 3. Now get all services belonging to this stack
	f := filters.NewArgs()
	f.Add("label", fmt.Sprintf("com.docker.stack.namespace=%s", stackName))
	services, err := c.ServiceList(ctx, types.ServiceListOptions{Filters: f})
	if err != nil {
		fmt.Println("failed to list services for stack:", stackName, ":", err)
		return nil
	}

	// 4. Filter only those running on this node
	var filtered []StackService
	for _, svc := range services {
		if _, ok := nodeServiceIDs[svc.ID]; ok {
			filtered = append(filtered, StackService{
				NodeID:      nodeID,
				StackName:   stackName,
				ServiceName: svc.Spec.Name,
			})
		}
	}

	sort.Slice(filtered, func(i, j int) bool {
		return filtered[i].ServiceName < filtered[j].ServiceName
	})

	return filtered
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
