package docker

import (
	"fmt"
	"sort"
)

type StackService struct {
	NodeID      string
	StackName   string
	ServiceName string
}

func GetStacks(nodeID string) []StackService {
	if nodeID == "" {
		return GetAllStacks()
	}
	return GetNodeStacks(nodeID)
}

func GetNodeStacks(nodeID string) []StackService {
	taskNames, err := GetNodeTaskNames(nodeID)
	if err != nil {
		return []StackService{}
	}

	serviceNames := extractServiceNames(taskNames)
	if len(serviceNames) == 0 {
		return []StackService{}
	}

	nameToID, err := GetServiceNameToIDMap()
	if err != nil {
		return []StackService{}
	}

	var serviceIDs []string
	for name := range serviceNames {
		if id, ok := nameToID[name]; ok {
			serviceIDs = append(serviceIDs, id)
		}
	}
	if len(serviceIDs) == 0 {
		return []StackService{}
	}

	stackServices, err := inspectStackServices(serviceIDs)
	if err != nil {
		return []StackService{}
	}

	// Translate node ID -> hostname
	idToName, _ := GetNodeIDToHostnameMap() // ignore map error; hostname is optional
	nodeName := nodeID
	if hn, ok := idToName[nodeID]; ok && hn != "" {
		nodeName = hn
	}

	for i := range stackServices {
		stackServices[i].NodeID = nodeName
	}

	sort.Slice(stackServices, func(i, j int) bool {
		if stackServices[i].StackName == stackServices[j].StackName {
			if stackServices[i].ServiceName == stackServices[j].ServiceName {
				return stackServices[i].NodeID < stackServices[j].NodeID
			}
			return stackServices[i].ServiceName < stackServices[j].ServiceName
		}
		return stackServices[i].StackName < stackServices[j].StackName
	})

	return stackServices
}

func GetAllStacks() []StackService {
	nodeIDs, err := GetNodeIDs()
	if err != nil {
		fmt.Println(err)
		return nil
	}

	idToName, _ := GetNodeIDToHostnameMap() // 🟢 ignore error if minor

	allStacks := make(map[string]StackService)
	for _, nodeID := range nodeIDs {
		nodeStacks := GetNodeStacks(nodeID)
		for _, s := range nodeStacks {
			if hostname, ok := idToName[s.NodeID]; ok {
				s.NodeID = hostname // 🟢 replace ID with hostname
			}
			key := s.StackName + "|" + s.ServiceName + "|" + s.NodeID
			allStacks[key] = s
		}
	}

	var uniqueStacks []StackService
	for _, s := range allStacks {
		uniqueStacks = append(uniqueStacks, s)
	}

	sort.Slice(uniqueStacks, func(i, j int) bool {
		if uniqueStacks[i].StackName == uniqueStacks[j].StackName {
			if uniqueStacks[i].NodeID == uniqueStacks[j].NodeID {
				return uniqueStacks[i].ServiceName < uniqueStacks[j].ServiceName
			}
			return uniqueStacks[i].NodeID < uniqueStacks[j].NodeID
		}
		return uniqueStacks[i].StackName < uniqueStacks[j].StackName
	})

	return uniqueStacks
}
