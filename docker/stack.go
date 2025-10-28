package docker

import (
	"context"
	"fmt"
	"sort"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

// GetStacks returns stacks across all nodes if nodeID is empty,
// or stacks only for the given node.
func GetStacks(nodeID string) []StackService {
	if nodeID == "" {
		return getStacksAllNodes()
	}
	return getStacksForNode(nodeID)
}

// getStacksForNode retrieves stack services for a specific node via the Docker SDK.
func getStacksForNode(nodeID string) []StackService {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Println("failed to init docker client:", err)
		return nil
	}
	defer cli.Close()

	// Filter tasks that belong to this node
	taskFilter := filters.NewArgs()
	taskFilter.Add("node", nodeID)
	tasks, err := cli.TaskList(ctx, types.TaskListOptions{Filters: taskFilter})
	if err != nil {
		fmt.Println("failed to list tasks for node", nodeID, ":", err)
		return nil
	}

	// Collect unique service IDs
	serviceIDs := make(map[string]struct{})
	for _, t := range tasks {
		if t.ServiceID != "" {
			serviceIDs[t.ServiceID] = struct{}{}
		}
	}

	if len(serviceIDs) == 0 {
		return nil
	}

	// Inspect each service to get stack info
	var stackServices []StackService
	for id := range serviceIDs {
		svc, _, err := cli.ServiceInspectWithRaw(ctx, id, types.ServiceInspectOptions{})
		if err != nil {
			continue
		}

		stackName := svc.Spec.Labels["com.docker.stack.namespace"]
		if stackName == "" {
			stackName = "(no-stack)"
		}

		stackServices = append(stackServices, StackService{
			NodeID:      resolveHostname(nodeID),
			StackName:   stackName,
			ServiceName: svc.Spec.Name,
		})
	}

	sortStackServices(stackServices)
	return stackServices
}

// getStacksAllNodes retrieves all stacks across all nodes via the Docker SDK.
func getStacksAllNodes() []StackService {
	ctx := context.Background()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		fmt.Println("failed to init docker client:", err)
		return nil
	}
	defer cli.Close()

	tasks, err := cli.TaskList(ctx, types.TaskListOptions{})
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
		svc, _, err := cli.ServiceInspectWithRaw(ctx, id, types.ServiceInspectOptions{})
		if err != nil {
			continue
		}

		stackName := svc.Spec.Labels["com.docker.stack.namespace"]
		if stackName == "" {
			stackName = "(no-stack)"
		}

		nodeName := resolveHostname(svc.Spec.Name) // fallback, if node info unavailable

		key := fmt.Sprintf("%s|%s|%s", stackName, svc.Spec.Name, nodeName)
		if _, exists := seen[key]; exists {
			continue
		}
		seen[key] = struct{}{}

		all = append(all, StackService{
			NodeID:      nodeName,
			StackName:   stackName,
			ServiceName: svc.Spec.Name,
		})
	}

	sortStackServices(all)
	return all
}

// --- helpers ---

// sortStackServices sorts stack services by stack, then node, then service name.
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
