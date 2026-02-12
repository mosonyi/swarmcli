// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

// Package features provides a simple feature flag registry.
// The OSS build has no features enabled. A pro build calls
// Enable() from init() after license validation.
package features

import "maps"

var enabled = map[string]bool{}

// Enable marks a feature as enabled.
func Enable(name string) { enabled[name] = true }

// Disable removes a feature flag.
func Disable(name string) { delete(enabled, name) }

// IsEnabled reports whether a feature is enabled.
func IsEnabled(name string) bool { return enabled[name] }

// All returns a copy of all enabled feature flags.
func All() map[string]bool {
	out := make(map[string]bool, len(enabled))
	maps.Copy(out, enabled)
	return out
}
