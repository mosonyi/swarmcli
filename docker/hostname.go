// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"
	"fmt"
	"sync"
	"time"
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
		nodeCacheMu.Lock()
		defer nodeCacheMu.Unlock()
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		nodeCacheErr = refreshNodeCacheLocked(ctx)
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

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := refreshNodeCacheLocked(ctx); err != nil {
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
// Caller must hold nodeCacheMu write lock.
func refreshNodeCacheLocked(ctx context.Context) error {
	names, err := GetNodeIDToHostnameMapFromDocker(ctx)
	if err != nil {
		return fmt.Errorf("refreshNodeCacheLocked: %w", err)
	}

	nodeCache = names
	return nil
}
