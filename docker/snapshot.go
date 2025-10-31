package docker

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/docker/docker/api/types"
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

	nodes, err := c.NodeList(ctx, types.NodeListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing nodes: %w", err)
	}

	services, err := c.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}

	tasks, err := c.TaskList(ctx, types.TaskListOptions{})
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
