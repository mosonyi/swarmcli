package stackservicesview

import (
	"fmt"
	"swarmcli/docker"
)

// LoadEntries loads all stacks and their services for a given node,
// including per-node replica counts like "(2/3 replicas)".
func LoadEntries(nodeID string) []ServiceEntry {
	stacks := docker.GetStacks(nodeID)
	var entries []ServiceEntry
	serviceMap, _ := docker.GetServiceNameToIDMap()

	for _, stack := range stacks {
		services := docker.GetServicesInStackOnNode(stack.Name, nodeID)
		for _, s := range services {
			displayName := s.ServiceName
			if s.ReplicasTotal > 0 { // only show counts if known
				displayName = fmt.Sprintf("%s (%d/%d replicas)", s.ServiceName, s.ReplicasOnNode, s.ReplicasTotal)
			}

			entries = append(entries, ServiceEntry{
				StackName:   stack.Name,
				ServiceName: displayName,
				ServiceID:   serviceMap[s.ServiceName],
			})
		}
	}

	return entries
}
