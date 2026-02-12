// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package features

import "testing"

func TestEnableAndIsEnabled(t *testing.T) {
	Enable("shell")
	if !IsEnabled("shell") {
		t.Error("expected shell to be enabled")
	}
}

func TestIsEnabled_Nonexistent(t *testing.T) {
	if IsEnabled("nonexistent") {
		t.Error("expected nonexistent to be disabled")
	}
}

func TestDisable(t *testing.T) {
	Enable("rbac")
	Disable("rbac")
	if IsEnabled("rbac") {
		t.Error("expected rbac to be disabled after Disable()")
	}
}

func TestAll_ReturnsCopy(t *testing.T) {
	Enable("test-feature")
	all := All()
	if !all["test-feature"] {
		t.Error("expected test-feature in All()")
	}
	// Mutating the copy must not affect the registry.
	delete(all, "test-feature")
	if !IsEnabled("test-feature") {
		t.Error("mutating All() result must not affect registry")
	}
}
