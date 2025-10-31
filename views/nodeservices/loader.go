package nodeservicesview

import (
	"fmt"
	"sort"
	"swarmcli/docker"

	"github.com/docker/docker/api/types/swarm"
)

// LoadEntries returns services filtered by nodeID or stackName.
// If both are empty, returns all services.
func LoadEntries(nodeID, stackName string) []ServiceEntry {
	snap, err := docker.GetOrRefreshSnapshot()
	if err != nil {
		fmt.Println("failed to get snapshot:", err)
		return nil
	}

	var entries []ServiceEntry

	for _, svc := range snap.Services {
		// Determine stack name
		stack := svc.Spec.Labels["com.docker.stack.namespace"]
		if stack == "" {
			stack = "-"
		}

		// Apply stack filter if specified
		if stackName != "" && stack != stackName {
			continue
		}

		// Determine desired replicas from service spec
		var desired int
		if svc.Spec.Mode.Replicated != nil && svc.Spec.Mode.Replicated.Replicas != nil {
			desired = int(*svc.Spec.Mode.Replicated.Replicas)
		} else if svc.Spec.Mode.Global != nil {
			desired = len(snap.Nodes)
		}

		// Count running tasks
		onNode := 0
		for _, t := range snap.Tasks {
			if t.ServiceID != svc.ID {
				continue
			}
			if t.DesiredState != swarm.TaskStateRunning || t.Status.State != swarm.TaskStateRunning {
				continue
			}
			if nodeID != "" && t.NodeID == nodeID {
				onNode++
			} else if nodeID == "" {
				// stack view: count all nodes
				onNode++
			}
		}

		// Skip services that don't run on the node (only for node view)
		if nodeID != "" && onNode == 0 {
			continue
		}

		entries = append(entries, ServiceEntry{
			StackName:      stack,
			ServiceName:    svc.Spec.Name,
			ServiceID:      svc.ID,
			ReplicasOnNode: onNode,
			ReplicasTotal:  desired,
		})
	}

	// Sort by stack then service name
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].StackName == entries[j].StackName {
			return entries[i].ServiceName < entries[j].ServiceName
		}
		return entries[i].StackName < entries[j].StackName
	})

	return entries
}
