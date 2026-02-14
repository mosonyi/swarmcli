// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

//go:build integration
// +build integration

package docker_test

import (
	"strings"
	"swarmcli/docker"
	"testing"
)

func TestReconstructStackCompose(t *testing.T) {
	// This test assumes that the demo stack is already deployed
	// (done by test-setup/testenv.sh deploy)
	stackName := "demo"

	yaml, err := docker.ReconstructStackCompose(stackName)
	if err != nil {
		t.Fatalf("Failed to reconstruct stack compose: %v", err)
	}

	if yaml == "" {
		t.Fatal("Reconstructed YAML is empty")
	}

	// Verify YAML contains expected elements
	if !strings.Contains(yaml, "version:") {
		t.Error("Reconstructed YAML missing version field")
	}

	if !strings.Contains(yaml, "services:") {
		t.Error("Reconstructed YAML missing services section")
	}

	// Check for known service names from test-stack.yml
	// (test-stack contains whoami and whoami_single services)
	if !strings.Contains(yaml, "whoami") {
		t.Error("Reconstructed YAML missing expected service 'whoami'")
	}

	t.Logf("Successfully reconstructed stack YAML (%d bytes)", len(yaml))
}

func TestReconstructStackCompose_NonExistentStack(t *testing.T) {
	stackName := "non-existent-stack-xyz123"

	_, err := docker.ReconstructStackCompose(stackName)
	if err == nil {
		t.Fatal("Expected error for non-existent stack, got nil")
	}

	if !strings.Contains(err.Error(), "no services found") && !strings.Contains(err.Error(), "failed to list services") {
		t.Errorf("Expected 'no services found' or 'failed to list services' error, got: %v", err)
	}
}
