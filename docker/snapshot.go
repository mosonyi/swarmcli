package docker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/api/types/swarm"
)

type NodeEntry struct {
	ID       string
	Hostname string
	Role     string
	State    string
	Manager  bool
	Addr     string
}

// StackEntry is a lightweight representation of a Docker stack,
// used for display and cached in SwarmSnapshot.
type StackEntry struct {
	Name         string
	ServiceCount int
	NodeCount    int
}

// SwarmSnapshot contains the in-memory swarm state.
type SwarmSnapshot struct {
	Nodes    []swarm.Node
	Services []swarm.Service
	Tasks    []swarm.Task
	Fetched  time.Time
}

var (
	snapshotMu sync.RWMutex
	snapshot   *SwarmSnapshot
)

// cacheTTL controls how long we reuse the snapshot before refreshing.
const cacheTTL = time.Minute

// GetSnapshot returns the cached snapshot if it's still valid.
func GetSnapshot() *SwarmSnapshot {
	snapshotMu.RLock()
	defer snapshotMu.RUnlock()
	return snapshot
}

// SetSnapshot replaces the cached snapshot (useful for manual refresh).
func SetSnapshot(s *SwarmSnapshot) {
	snapshotMu.Lock()
	defer snapshotMu.Unlock()
	snapshot = s
}

// RefreshSnapshot fetches all swarm data (nodes, services, tasks) at once
// and updates the global cache.
func RefreshSnapshot() (*SwarmSnapshot, error) {
	c, err := GetClient()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	defer c.Close()

	ctx := context.Background()

	nodes, err := c.NodeList(ctx, swarm.NodeListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing nodes: %w", err)
	}

	services, err := c.ServiceList(ctx, swarm.ServiceListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}

	tasks, err := c.TaskList(ctx, swarm.TaskListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing tasks: %w", err)
	}

	snap := &SwarmSnapshot{
		Nodes:    nodes,
		Services: services,
		Tasks:    tasks,
		Fetched:  time.Now(),
	}

	SetSnapshot(snap)
	return snap, nil
}

// GetOrRefreshSnapshot returns the current snapshot, refreshing it
// if the cache is empty or too old.
func GetOrRefreshSnapshot() (*SwarmSnapshot, error) {
	snapshotMu.RLock()
	s := snapshot
	snapshotMu.RUnlock()

	if s == nil || time.Since(s.Fetched) > cacheTTL {
		return RefreshSnapshot()
	}
	return s, nil
}

// ToNodeEntries converts the full nodes into display-friendly entries.
func (s SwarmSnapshot) ToNodeEntries() []NodeEntry {
	nodes := make([]NodeEntry, len(s.Nodes))
	for i, n := range s.Nodes {
		nodes[i] = NodeEntry{
			ID:       n.ID,
			Hostname: n.Description.Hostname,
			Role:     string(n.Spec.Role),
			State:    string(n.Status.State),
			Manager:  n.ManagerStatus != nil,
			Addr:     n.Status.Addr,
		}
	}
	return nodes
}

// ToStackEntries aggregates services by stack name and produces StackEntry slices.
func (s SwarmSnapshot) ToStackEntries() []StackEntry {
	stackMap := make(map[string]*StackEntry)

	for _, svc := range s.Services {
		// Docker stacks mark services with com.docker.stack.namespace
		if stackName, ok := svc.Spec.Labels["com.docker.stack.namespace"]; ok {
			entry, exists := stackMap[stackName]
			if !exists {
				stackMap[stackName] = &StackEntry{Name: stackName, ServiceCount: 1}
			} else {
				entry.ServiceCount++
			}
		}
	}

	// Optionally count how many nodes run tasks of that stack
	for _, task := range s.Tasks {
		if task.ServiceID == "" {
			continue
		}
		// Find stack name from the service
		var stackName string
		for _, svc := range s.Services {
			if svc.ID == task.ServiceID {
				stackName = svc.Spec.Labels["com.docker.stack.namespace"]
				break
			}
		}
		if stackName == "" {
			continue
		}
		entry := stackMap[stackName]
		if entry != nil {
			entry.NodeCount++ // Each task counts as one node slot, could refine if needed
		}
	}

	stacks := make([]StackEntry, 0, len(stackMap))
	for _, e := range stackMap {
		stacks = append(stacks, *e)
	}

	return stacks
}

// FindService looks up a service by ID in the snapshot.
func (s *SwarmSnapshot) FindService(serviceID string) *swarm.Service {
	for i := range s.Services {
		if s.Services[i].ID == serviceID {
			return &s.Services[i]
		}
	}
	return nil
}

// FindServiceByName looks up a service by its name in the snapshot.
func (s *SwarmSnapshot) FindServiceByName(name string) *swarm.Service {
	for i := range s.Services {
		if s.Services[i].Spec.Name == name {
			return &s.Services[i]
		}
	}
	return nil
}
