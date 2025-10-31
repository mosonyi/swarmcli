package nodeservicesview

import (
	"fmt"
	"sort"
	"swarmcli/docker"

	"github.com/docker/docker/api/types/swarm"
)

func LoadEntries(nodeID, stackName string) []ServiceEntry {
	snap, err := docker.GetOrRefreshSnapshot()
	if err != nil {
		fmt.Println("failed to get snapshot:", err)
		return nil
	}

	var entries []ServiceEntry

	// Count currently running tasks per service (optionally filtered by node)
	runningTasks := map[string]int{}
	for _, t := range snap.Tasks {
		if t.DesiredState != swarm.TaskStateRunning || t.Status.State != swarm.TaskStateRunning {
			continue
		}
		if nodeID != "" && t.NodeID != nodeID {
			continue
		}
		runningTasks[t.ServiceID]++
	}

	for _, svc := range snap.Services {
		stack := svc.Spec.Labels["com.docker.stack.namespace"]
		if stack == "" {
			stack = "-"
		}
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

		onNode := runningTasks[svc.ID] // could be 0 if no tasks currently running

		entries = append(entries, ServiceEntry{
			StackName:      stack,
			ServiceName:    svc.Spec.Name,
			ServiceID:      svc.ID,
			ReplicasOnNode: onNode,
			ReplicasTotal:  desired,
		})
	}

	// Sort by stack then service
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].StackName == entries[j].StackName {
			return entries[i].ServiceName < entries[j].ServiceName
		}
		return entries[i].StackName < entries[j].StackName
	})

	return entries
}
