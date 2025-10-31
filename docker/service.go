package docker

import (
	"context"
	"fmt"
	"sort"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/swarm"
)

type StackService struct {
	NodeID         string
	StackName      string
	ServiceName    string
	ServiceID      string
	ReplicasOnNode int
	ReplicasTotal  int
}

// GetServicesInStackOnNode returns services in a stack that have tasks on the given node,
// along with per-node replica counts (e.g. 2/3 replicas).
func GetServicesInStackOnNode(stackName, nodeID string) []StackService {
	c, err := GetClient()
	if err != nil {
		fmt.Println("failed to init docker client:", err)
		return nil
	}
	defer c.Close()

	ctx := context.Background()

	// 1. List all tasks across the swarm for the stack
	taskFilter := filters.NewArgs()
	taskFilter.Add("label", fmt.Sprintf("com.docker.stack.namespace=%s", stackName))
	tasks, err := c.TaskList(ctx, types.TaskListOptions{Filters: taskFilter})
	if err != nil {
		fmt.Println("failed to list tasks for stack:", stackName, ":", err)
		return nil
	}

	// 2. Count replicas by service and by node
	type count struct {
		total  int
		onNode int
	}
	replicaCounts := make(map[string]*count)

	for _, t := range tasks {
		if t.DesiredState != swarm.TaskStateRunning {
			continue
		}
		svcID := t.ServiceID
		if svcID == "" {
			continue
		}
		c := replicaCounts[svcID]
		if c == nil {
			c = &count{}
			replicaCounts[svcID] = c
		}
		c.total++
		if t.NodeID == nodeID {
			c.onNode++
		}
	}

	// 3. List all services belonging to the stack
	f := filters.NewArgs()
	f.Add("label", fmt.Sprintf("com.docker.stack.namespace=%s", stackName))
	services, err := c.ServiceList(ctx, types.ServiceListOptions{Filters: f})
	if err != nil {
		fmt.Println("failed to list services for stack:", stackName, ":", err)
		return nil
	}

	// 4. Filter services that actually have tasks on this node
	var filtered []StackService
	for _, svc := range services {
		if c, ok := replicaCounts[svc.ID]; ok && c.onNode > 0 {
			filtered = append(filtered, StackService{
				NodeID:         nodeID,
				StackName:      stackName,
				ServiceName:    svc.Spec.Name,
				ServiceID:      svc.ID,
				ReplicasOnNode: c.onNode,
				ReplicasTotal:  c.total,
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
