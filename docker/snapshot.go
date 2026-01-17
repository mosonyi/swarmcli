package docker

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/docker/docker/api/types/swarm"
)

type NodeEntry struct {
	ID           string
	Version      string
	Hostname     string
	Role         string
	State        string
	Availability string
	Manager      bool
	Addr         string
	Labels       map[string]string
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
	// refreshInProgress indicates whether a background refresh goroutine is running.
	refreshInProgress int32
)

// cacheTTL controls how long we reuse the snapshot before refreshing.
const cacheTTL = 10 * time.Second

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

// InvalidateSnapshot clears the cached snapshot, forcing a fresh fetch on next access.
// This should be called after a Docker context switch.
func InvalidateSnapshot() {
	snapshotMu.Lock()
	defer snapshotMu.Unlock()
	snapshot = nil
}

// RefreshSnapshot fetches all swarm data (nodes, services, tasks) at once
// and updates the global cache.
func RefreshSnapshot() (*SwarmSnapshot, error) {
	c, err := GetClient()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}
	defer closeCli(c)

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

// RefreshSnapshotAsync triggers a background refresh if one is not already running.
// It returns immediately.
func RefreshSnapshotAsync() {
	if !atomic.CompareAndSwapInt32(&refreshInProgress, 0, 1) {
		// already refreshing
		return
	}

	go func() {
		defer atomic.StoreInt32(&refreshInProgress, 0)
		// Run a refresh and ignore the error; the cached snapshot will be updated on success.
		_, _ = RefreshSnapshot()
	}()
}

// TriggerRefreshIfNeeded will check the cache TTL and start a background refresh
// if the snapshot is empty or stale.
func TriggerRefreshIfNeeded() {
	snapshotMu.RLock()
	s := snapshot
	snapshotMu.RUnlock()

	if s == nil || time.Since(s.Fetched) > cacheTTL {
		RefreshSnapshotAsync()
	}
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
		ver := "-"
		if n.Description.Engine.EngineVersion != "" {
			ver = n.Description.Engine.EngineVersion
		}
		avail := string(n.Spec.Availability)
		if avail == "" {
			avail = "active"
		}
		// A node is a manager if either ManagerStatus is populated OR the role is explicitly set to manager
		isManager := n.ManagerStatus != nil || n.Spec.Role == swarm.NodeRoleManager
		nodes[i] = NodeEntry{
			ID:           n.ID,
			Version:      ver,
			Hostname:     n.Description.Hostname,
			Role:         string(n.Spec.Role),
			State:        string(n.Status.State),
			Availability: avail,
			Manager:      isManager,
			Addr:         n.Status.Addr,
			Labels:       n.Spec.Labels,
		}
	}

	// 🔠 Sort alphabetically by hostname
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].Hostname < nodes[j].Hostname
	})

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

	// ---- 🔠 Sort alphabetically by stack name ----
	sort.Slice(stacks, func(i, j int) bool {
		return stacks[i].Name < stacks[j].Name
	})

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
