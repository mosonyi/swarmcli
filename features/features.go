// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

// Package features provides a simple feature flag registry.
// The base build has no features enabled. Extension builds
// call Enable() from init() to activate additional features.
package features

import (
	"maps"
	"sync"
)

// mu guards enabled: it is written from the Bubble Tea update goroutine
// (ApplyLicenseState on startup / license / swarm change) and read from
// tea.Cmd goroutines (feature-gated data loaders), so unsynchronized access
// would be a concurrent map read/write.
var (
	mu      sync.RWMutex
	enabled = map[string]bool{}
)

// Enable marks a feature as enabled.
func Enable(name string) {
	mu.Lock()
	defer mu.Unlock()
	enabled[name] = true
}

// Disable removes a feature flag.
func Disable(name string) {
	mu.Lock()
	defer mu.Unlock()
	delete(enabled, name)
}

// IsEnabled reports whether a feature is enabled.
func IsEnabled(name string) bool {
	mu.RLock()
	defer mu.RUnlock()
	return enabled[name]
}

// All returns a copy of all enabled feature flags.
func All() map[string]bool {
	mu.RLock()
	defer mu.RUnlock()
	out := make(map[string]bool, len(enabled))
	maps.Copy(out, enabled)
	return out
}
