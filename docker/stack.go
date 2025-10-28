package docker

import (
	"context"
	"fmt"
	"log"
	"sort"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	swarmTypes "github.com/docker/docker/api/types/swarm"
)

// Stack represents a unique Docker stack.
type Stack struct {
	Name string
}

// GetStacks returns all unique stacks, optionally filtered by node.
func GetStacks(nodeID string) []Stack {
	c, err := GetClient()

	log.Println("Getting Stacks:")

	if err != nil {
		fmt.Println("failed to init docker client:", err)
		return nil
	}
	defer c.Close()

	ctx := context.Background()

	var services []swarmTypes.Service
	if nodeID == "" {
		// Fast path: list all services once.
		services, err = c.ServiceList(ctx, types.ServiceListOptions{})
		if err != nil {
			log.Println("failed to list services:", err)
			return nil
		}
	} else {
		// Slower path: only keep services that have tasks on the given node.
		tasks, err := c.TaskList(ctx, types.TaskListOptions{
			Filters: filters.NewArgs(filters.Arg("node", nodeID)),
		})
		if err != nil {
			log.Println("failed to list tasks for node:", nodeID, ":", err)
			return nil
		}

		serviceIDs := make(map[string]struct{})
		for _, t := range tasks {
			if t.ServiceID != "" {
				serviceIDs[t.ServiceID] = struct{}{}
			}
		}

		for id := range serviceIDs {
			svc, _, err := c.ServiceInspectWithRaw(ctx, id, types.ServiceInspectOptions{})
			if err == nil {
				services = append(services, svc)
			}
		}
	}

	// Deduplicate stacks
	stackSet := make(map[string]struct{})
	for _, svc := range services {
		stack := svc.Spec.Labels["com.docker.stack.namespace"]
		if stack == "" {
			stack = "(no-stack)"
		}
		stackSet[stack] = struct{}{}
	}

	stacks := make([]Stack, 0, len(stackSet))
	for name := range stackSet {
		stacks = append(stacks, Stack{Name: name})
	}

	sort.Slice(stacks, func(i, j int) bool {
		return stacks[i].Name < stacks[j].Name
	})
	return stacks
}
