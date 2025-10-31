package stackservicesview

import (
	"swarmcli/docker"
)

// LoadEntries loads all stacks and their services for a given node,
// including per-node replica counts.
func LoadEntries(nodeID string) []ServiceEntry {
	stacks := docker.GetStacks(nodeID)
	var entries []ServiceEntry
	serviceMap, _ := docker.GetServiceNameToIDMap()

	for _, stack := range stacks {
		services := docker.GetServicesInStackOnNode(stack.Name, nodeID)
		for _, s := range services {
			entries = append(entries, ServiceEntry{
				StackName:      stack.Name,
				ServiceName:    s.ServiceName,
				ServiceID:      serviceMap[s.ServiceName],
				ReplicasOnNode: s.ReplicasOnNode,
				ReplicasTotal:  s.ReplicasTotal,
			})
		}
	}
	return entries
}
