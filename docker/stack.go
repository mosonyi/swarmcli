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
	Name         string
	ServiceCount int
}

func GetStacks(nodeID string) []Stack {
	c, err := GetClient()
	if err != nil {
		fmt.Println("failed to init docker client:", err)
		return nil
	}
	defer c.Close()

	ctx := context.Background()

	var services []swarmTypes.Service
	if nodeID == "" {
		services, err = c.ServiceList(ctx, types.ServiceListOptions{})
		if err != nil {
			log.Println("failed to list services:", err)
			return nil
		}
	} else {
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

	// Count services per stack
	stackCount := make(map[string]int)
	for _, svc := range services {
		stack := svc.Spec.Labels["com.docker.stack.namespace"]
		if stack == "" {
			stack = "(no-stack)"
		}
		stackCount[stack]++
	}

	stacks := make([]Stack, 0, len(stackCount))
	for name, count := range stackCount {
		stacks = append(stacks, Stack{
			Name:         name,
			ServiceCount: count,
		})
	}

	sort.Slice(stacks, func(i, j int) bool {
		return stacks[i].Name < stacks[j].Name
	})
	return stacks
}
