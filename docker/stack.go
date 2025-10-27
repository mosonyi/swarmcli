package docker

import (
	"fmt"
	"sort"
	"sync"
)

// StackService represents a single service inside a stack on a node.
type StackService struct {
	NodeID      string
	StackName   string
	ServiceName string
}

var (
	nodeNameCache     map[string]string
	nodeNameCacheOnce sync.Once
	nodeCacheMu       sync.RWMutex
)

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

	initHostnameCache()

	allStacks := make(map[string]StackService)
	for _, nodeID := range nodeIDs {
		nodeStacks := getNodeStacks(nodeID)
		for _, s := range nodeStacks {
			if hostname := resolveHostname(s.NodeID); hostname != "" {
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

// resolveHostname translates a node ID to its hostname using a cached map.
func resolveHostname(nodeID string) string {
	initHostnameCache()
	nodeCacheMu.RLock()
	defer nodeCacheMu.RUnlock()
	if name, ok := nodeNameCache[nodeID]; ok && name != "" {
		return name
	}
	return nodeID
}

// initHostnameCache loads the node ID → hostname map once per runtime.
func initHostnameCache() {
	nodeNameCacheOnce.Do(func() {
		refreshNodeCacheInternal()
	})
}

// RefreshNodeCache forcibly refreshes the node hostname cache.
// This can be triggered when nodes change (join/leave/rename).
func RefreshNodeCache() error {
	nodeCacheMu.Lock()
	defer nodeCacheMu.Unlock()
	return refreshNodeCacheInternal()
}

func refreshNodeCacheInternal() error {
	idToName, err := GetNodeIDToHostnameMap()
	if err != nil {
		nodeNameCache = make(map[string]string)
		return err
	}
	nodeNameCache = idToName
	return nil
}

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
