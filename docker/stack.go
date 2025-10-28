package docker

import (
	"context"
	"fmt"
	"sort"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
)

// GetStacks returns stacks across all nodes if nodeID is empty,
// or stacks only for the given node.
func GetStacks(nodeID string) []StackService {
	if nodeID == "" {
		return getStacksAllNodes()
	}
	return getStacksForNode(nodeID)
}

func getStacksForNode(nodeID string) []StackService {
	c, err := GetClient()
	if err != nil {
		fmt.Println("failed to init docker client:", err)
		return nil
	}
	defer c.Close()

	ctx := context.Background()
	taskFilter := filters.NewArgs()
	taskFilter.Add("node", nodeID)

	tasks, err := c.TaskList(ctx, types.TaskListOptions{Filters: taskFilter})
	if err != nil {
		fmt.Println("failed to list tasks for node", nodeID, ":", err)
		return nil
	}

	serviceIDs := make(map[string]struct{})
	for _, t := range tasks {
		if t.ServiceID != "" {
			serviceIDs[t.ServiceID] = struct{}{}
		}
	}

	var stackServices []StackService
	for id := range serviceIDs {
		svc, _, err := c.ServiceInspectWithRaw(ctx, id, types.ServiceInspectOptions{})
		if err != nil {
			continue
		}
		stack := svc.Spec.Labels["com.docker.stack.namespace"]
		if stack == "" {
			stack = "(no-stack)"
		}
		stackServices = append(stackServices, StackService{
			NodeID:      resolveHostname(nodeID),
			StackName:   stack,
			ServiceName: svc.Spec.Name,
		})
	}

	sortStackServices(stackServices)
	return stackServices
}

func getStacksAllNodes() []StackService {
	c, err := GetClient()
	if err != nil {
		fmt.Println("failed to init docker client:", err)
		return nil
	}
	defer c.Close()

	ctx := context.Background()
	tasks, err := c.TaskList(ctx, types.TaskListOptions{})
	if err != nil {
		fmt.Println("failed to list all tasks:", err)
		return nil
	}

	serviceIDs := make(map[string]struct{})
	for _, t := range tasks {
		if t.ServiceID != "" {
			serviceIDs[t.ServiceID] = struct{}{}
		}
	}

	var all []StackService
	seen := make(map[string]struct{})

	for id := range serviceIDs {
		svc, _, err := c.ServiceInspectWithRaw(ctx, id, types.ServiceInspectOptions{})
		if err != nil {
			continue
		}
		stack := svc.Spec.Labels["com.docker.stack.namespace"]
		if stack == "" {
			stack = "(no-stack)"
		}

		nodeName := resolveHostname(svc.Spec.Name)
		key := fmt.Sprintf("%s|%s|%s", stack, svc.Spec.Name, nodeName)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		all = append(all, StackService{
			NodeID:      nodeName,
			StackName:   stack,
			ServiceName: svc.Spec.Name,
		})
	}

	sortStackServices(all)
	return all
}

func sortStackServices(stacks []StackService) {
	sort.Slice(stacks, func(i, j int) bool {
		a, b := stacks[i], stacks[j]
		if a.StackName != b.StackName {
			return a.StackName < b.StackName
		}
		if a.NodeID != b.NodeID {
			return a.NodeID < b.NodeID
		}
		return a.ServiceName < b.ServiceName
	})
}
