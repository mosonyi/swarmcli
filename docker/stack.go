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
		return nil
	}

	serviceNames := extractServiceNames(taskNames)
	if len(serviceNames) == 0 {
		return nil
	}

	nameToID, err := GetServiceNameToIDMap()
	if err != nil {
		return nil
	}

	var serviceIDs []string
	for name := range serviceNames {
		if id, ok := nameToID[name]; ok {
			serviceIDs = append(serviceIDs, id)
		}
	}
	if len(serviceIDs) == 0 {
		return nil
	}

	stackServices, err := inspectStackServices(serviceIDs)
	if err != nil {
		return nil
	}

	// 🟢 Fix: attach nodeID to each StackService
	for i := range stackServices {
		stackServices[i].NodeID = nodeID
	}

	sort.Slice(stackServices, func(i, j int) bool {
		return stackServices[i].StackName < stackServices[j].StackName
	})

	return stackServices
}

func GetAllStacks() []StackService {
	nodeIDs, err := GetNodeIDs()
	if err != nil {
		fmt.Println(err)
		return []StackService{}
	}

	allStacks := make(map[string]StackService)
	for _, nodeID := range nodeIDs {
		for _, s := range GetNodeStacks(nodeID) {
			key := s.StackName + "|" + s.ServiceName
			allStacks[key] = s
		}
	}

	uniqueStacks := make([]StackService, 0, len(allStacks))
	for _, s := range allStacks {
		uniqueStacks = append(uniqueStacks, s)
	}

	sort.Slice(uniqueStacks, func(i, j int) bool {
		if uniqueStacks[i].StackName == uniqueStacks[j].StackName {
			return uniqueStacks[i].ServiceName < uniqueStacks[j].ServiceName
		}
		return uniqueStacks[i].StackName < uniqueStacks[j].StackName
	})

	return uniqueStacks
}
