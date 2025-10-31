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
	type count struct{ total, onNode int }
	counts := map[string]*count{}

	// Count tasks per service
	for _, t := range snap.Tasks {
		if t.DesiredState != swarm.TaskStateRunning {
			continue
		}
		c := counts[t.ServiceID]
		if c == nil {
			c = &count{}
			counts[t.ServiceID] = c
		}
		c.total++
		if nodeID != "" && t.NodeID == nodeID {
			c.onNode++
		}
	}

	for _, svc := range snap.Services {
		svcStack := svc.Spec.Labels["com.docker.stack.namespace"]
		if svcStack == "" {
			svcStack = "-"
		}

		if stackName != "" && svcStack != stackName {
			continue
		}

		c := counts[svc.ID]
		if c == nil {
			continue
		}

		var replicasOnNode int
		if nodeID != "" {
			replicasOnNode = c.onNode
			// Skip services not running on this node
			if replicasOnNode == 0 {
				continue
			}
		} else {
			// Stack view: show all running tasks as "on node"
			replicasOnNode = c.total
		}

		entries = append(entries, ServiceEntry{
			StackName:      svcStack,
			ServiceName:    svc.Spec.Name,
			ServiceID:      svc.ID,
			ReplicasOnNode: replicasOnNode,
			ReplicasTotal:  c.total,
		})
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].StackName == entries[j].StackName {
			return entries[i].ServiceName < entries[j].ServiceName
		}
		return entries[i].StackName < entries[j].StackName
	})

	return entries
}
