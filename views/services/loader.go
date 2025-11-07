package servicesview

import (
	"sort"
	"swarmcli/docker"

	"github.com/docker/docker/api/types/swarm"
)

func LoadNodeServices(nodeID string) []ServiceEntry {
	snap, err := docker.GetOrRefreshSnapshot()
	if err != nil {
		l().Infof("failed to get snapshot:", err)
		return nil
	}

	var entries []ServiceEntry

	for _, svc := range snap.Services {
		stack, desired := getServiceStackAndDesired(svc, snap)

		// Count tasks running on this node
		onNode := countTasksForNode(svc.ID, nodeID, snap)

		// Skip if no tasks on this node
		if onNode == 0 {
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

	sortEntries(entries)
	return entries
}

func LoadStackServices(stackName string) []ServiceEntry {
	snap, err := docker.GetOrRefreshSnapshot()
	if err != nil {
		l().Infof("failed to get snapshot:", err)
		return nil
	}

	var entries []ServiceEntry

	for _, svc := range snap.Services {
		stack, desired := getServiceStackAndDesired(svc, snap)
		if stack != stackName {
			continue
		}

		// Count tasks on all nodes
		onNode := countTasksForNode(svc.ID, "", snap)

		entries = append(entries, ServiceEntry{
			StackName:      stack,
			ServiceName:    svc.Spec.Name,
			ServiceID:      svc.ID,
			ReplicasOnNode: onNode,
			ReplicasTotal:  desired,
		})
	}

	sortEntries(entries)
	return entries
}

// --- Helpers ---

// getServiceStackAndDesired returns the stack name and desired replicas for a service
func getServiceStackAndDesired(svc swarm.Service, snap *docker.SwarmSnapshot) (stack string, desired int) {
	stack = svc.Spec.Labels["com.docker.stack.namespace"]
	if stack == "" {
		stack = "-"
	}

	if svc.Spec.Mode.Replicated != nil && svc.Spec.Mode.Replicated.Replicas != nil {
		desired = int(*svc.Spec.Mode.Replicated.Replicas)
	} else if svc.Spec.Mode.Global != nil {
		desired = len(snap.Nodes)
	} else {
		// One-off task
		desired = 1
	}

	return
}

// countTasksForNode counts tasks for a service; if nodeID == "", counts across all nodes
func countTasksForNode(serviceID, nodeID string, snap *docker.SwarmSnapshot) int {
	count := 0
	for _, t := range snap.Tasks {
		if t.ServiceID != serviceID {
			continue
		}
		if t.DesiredState != swarm.TaskStateRunning {
			continue
		}
		if nodeID == "" || t.NodeID == nodeID {
			count++
		}
	}
	return count
}

// sortEntries sorts entries by stack name then service name
func sortEntries(entries []ServiceEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].StackName == entries[j].StackName {
			return entries[i].ServiceName < entries[j].ServiceName
		}
		return entries[i].StackName < entries[j].StackName
	})
}
