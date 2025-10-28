package docker

import (
	"fmt"
	"sync"
)

// Cache for nodeID → hostname lookups.
var (
	nodeCacheOnce sync.Once
	nodeCacheErr  error
	nodeCache     map[string]string
	nodeCacheMu   sync.RWMutex
)

// ensureHostnameCache lazily initializes the cache once per runtime.
func ensureHostnameCache() error {
	nodeCacheOnce.Do(func() {
		nodeCacheErr = refreshNodeCacheLocked()
	})
	return nodeCacheErr
}

// RefreshHostnameCache forcibly refreshes the cache (e.g. triggered by UI).
// Safe to call concurrently.
func RefreshHostnameCache() error {
	nodeCacheMu.Lock()
	defer nodeCacheMu.Unlock()

	// Reset the Once so ensureHostnameCache() can run again if needed.
	nodeCacheOnce = sync.Once{}

	if err := refreshNodeCacheLocked(); err != nil {
		nodeCacheErr = fmt.Errorf("manual hostname cache refresh failed: %w", err)
		return nodeCacheErr
	}

	nodeCacheErr = nil
	return nil
}

// GetNodeIDToHostnameMap returns a copy of the cached map.
// Automatically initializes the cache if needed.
func GetNodeIDToHostnameMap() (map[string]string, error) {
	if err := ensureHostnameCache(); err != nil {
		return nil, fmt.Errorf("failed to initialize hostname cache: %w", err)
	}

	nodeCacheMu.RLock()
	defer nodeCacheMu.RUnlock()

	cpy := make(map[string]string, len(nodeCache))
	for k, v := range nodeCache {
		cpy[k] = v
	}
	return cpy, nil
}

// refreshNodeCacheLocked updates the global cache map in-place.
// Caller must hold the write lock if not called from ensureHostnameCache().
func refreshNodeCacheLocked() error {
	names, err := GetNodeIDToHostnameMapFromDocker()
	if err != nil {
		return fmt.Errorf("refreshNodeCacheLocked: %w", err)
	}

	nodeCacheMu.Lock()
	defer nodeCacheMu.Unlock()
	nodeCache = names
	return nil
}
