package nodeservicesview

import (
	"fmt"
	"sort"
	"swarmcli/docker"

	"github.com/docker/docker/api/types/swarm"
)

func LoadEntries(nodeID string) []ServiceEntry {
	snap, err := docker.GetOrRefreshSnapshot()
	if err != nil {
		fmt.Println("failed to get snapshot:", err)
		return nil
	}

	// filter services/tasks for this node
	var entries []ServiceEntry
	type count struct{ total, onNode int }
	counts := map[string]*count{}

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
		if t.NodeID == nodeID {
			c.onNode++
		}
	}

	for _, svc := range snap.Services {
		stackName := svc.Spec.Labels["com.docker.stack.namespace"]
		if stackName == "" {
			stackName = "-"
		}
		if c, ok := counts[svc.ID]; ok && c.onNode > 0 {
			entries = append(entries, ServiceEntry{
				StackName:      stackName,
				ServiceName:    svc.Spec.Name,
				ServiceID:      svc.ID,
				ReplicasOnNode: c.onNode,
				ReplicasTotal:  c.total,
			})
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].StackName == entries[j].StackName {
			return entries[i].ServiceName < entries[j].ServiceName
		}
		return entries[i].StackName < entries[j].StackName
	})
	return entries
}
