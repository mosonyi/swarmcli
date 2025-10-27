package docker

import (
	"fmt"
	"sort"
)

// StackService represents a single service inside a stack on a node.
type StackService struct {
	NodeID      string // node hostname or ID
	StackName   string // stack namespace
	ServiceName string // service name
}

// GetStacks returns stacks across all nodes if nodeID is empty,
// or stacks only for the given node.
func GetStacks(nodeID string) []StackService {
	if nodeID == "" {
		return getStacksAllNodes()
	}
	return getStacksForNode(nodeID)
}

// getStacksForNode retrieves stack services for a specific node.
func getStacksForNode(nodeID string) []StackService {
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

// getStacksAllNodes retrieves all stacks across all nodes.
func getStacksAllNodes() []StackService {
	nodeIDs, err := GetNodeIDs()
	if err != nil {
		return nil
	}

	seen := make(map[string]struct{})
	var all []StackService

	for _, nodeID := range nodeIDs {
		for _, s := range getStacksForNode(nodeID) {
			s.NodeID = resolveHostname(s.NodeID)

			key := fmt.Sprintf("%s|%s|%s", s.StackName, s.ServiceName, s.NodeID)
			if _, exists := seen[key]; exists {
				continue
			}
			seen[key] = struct{}{}
			all = append(all, s)
		}
	}

	sortStackServices(all)
	return all
}

// --- helpers ---

// resolveServiceIDs maps task names → service IDs.
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

// sortStackServices sorts stack services by stack, then node, then service name.
func sortStackServices(stacks []StackService) {
	sort.Slice(stacks, func(i, j int) bool {
		a, b := stacks[i], stacks[j]
		if a.StackName != b.StackName {
			return a.StackName < b.StackName
		}
		if a.NodeID != b.NodeID {
			return a.NodeID < b.NodeID
		}
		return a.ServiceName < b.ServiceName
	})
}
