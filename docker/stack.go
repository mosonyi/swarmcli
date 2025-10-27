package docker

import (
	"fmt"
	"sort"
)

// StackService represents a single service inside a stack on a node.
type StackService struct {
	NodeID      string
	StackName   string
	ServiceName string
}

// GetStacks returns stacks across all nodes if nodeID is empty,
// or stacks only for the given node.
func GetStacks(nodeID string) []StackService {
	if nodeID == "" {
		return getAllStacks()
	}
	return getNodeStacks(nodeID)
}

// getNodeStacks retrieves stack services for a specific node.
func getNodeStacks(nodeID string) []StackService {
	taskNames, err := GetNodeTaskNames(nodeID)
	if err != nil {
		return nil
	}

	serviceIDs, err := resolveServiceIDs(taskNames)
	if err != nil || len(serviceIDs) == 0 {
		return nil
	}

	stackServices, err := inspectStackServices(serviceIDs)
	if err != nil {
		return nil
	}

	nodeName := resolveHostname(nodeID)
	for i := range stackServices {
		stackServices[i].NodeID = nodeName
	}

	sortStackServices(stackServices)
	return stackServices
}

// getAllStacks retrieves all stacks across all nodes.
func getAllStacks() []StackService {
	nodeIDs, err := GetNodeIDs()
	if err != nil {
		fmt.Println("error getting nodes:", err)
		return nil
	}

	idToName, _ := GetNodeIDToHostnameMap()

	allStacks := make(map[string]StackService)
	for _, nodeID := range nodeIDs {
		nodeStacks := getNodeStacks(nodeID)
		for _, s := range nodeStacks {
			if hostname, ok := idToName[s.NodeID]; ok {
				s.NodeID = hostname
			}
			key := fmt.Sprintf("%s|%s|%s", s.StackName, s.ServiceName, s.NodeID)
			allStacks[key] = s
		}
	}

	uniqueStacks := make([]StackService, 0, len(allStacks))
	for _, s := range allStacks {
		uniqueStacks = append(uniqueStacks, s)
	}

	sortStackServices(uniqueStacks)
	return uniqueStacks
}

// --- helpers ---

// resolveServiceIDs converts task names into Docker service IDs.
func resolveServiceIDs(taskNames []string) ([]string, error) {
	serviceNames := extractServiceNames(taskNames)
	if len(serviceNames) == 0 {
		return nil, nil
	}

	nameToID, err := GetServiceNameToIDMap()
	if err != nil {
		return nil, err
	}

	serviceIDs := make([]string, 0, len(serviceNames))
	for name := range serviceNames {
		if id, ok := nameToID[name]; ok {
			serviceIDs = append(serviceIDs, id)
		}
	}
	return serviceIDs, nil
}

// resolveHostname translates a node ID to its hostname (if available).
func resolveHostname(nodeID string) string {
	idToName, err := GetNodeIDToHostnameMap()
	if err == nil {
		if name, ok := idToName[nodeID]; ok && name != "" {
			return name
		}
	}
	return nodeID
}

// sortStackServices sorts by StackName → NodeID → ServiceName.
func sortStackServices(stacks []StackService) {
	sort.Slice(stacks, func(i, j int) bool {
		a, b := stacks[i], stacks[j]
		switch {
		case a.StackName != b.StackName:
			return a.StackName < b.StackName
		case a.NodeID != b.NodeID:
			return a.NodeID < b.NodeID
		default:
			return a.ServiceName < b.ServiceName
		}
	})
}
