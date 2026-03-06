// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

//go:build integration
// +build integration

package docker_test

import (
	"strings"
	"swarmcli/docker"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
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

	// Verify volumes section
	if !strings.Contains(yaml, "volumes:") {
		t.Error("Reconstructed YAML missing volumes section")
	}
	if !strings.Contains(yaml, "whoami_data:") {
		t.Error("Reconstructed YAML missing volume 'whoami_data' (should be stripped of 'demo_' prefix)")
	}

	// Verify networks section
	if !strings.Contains(yaml, "networks:") {
		t.Error("Reconstructed YAML missing networks section")
	}
	if !strings.Contains(yaml, "backend:") {
		t.Error("Reconstructed YAML missing network 'backend' (should be stripped of 'demo_' prefix)")
	}

	// Stack-managed networks should NOT be marked external
	if strings.Contains(yaml, "external: true") {
		t.Error("Stack-managed network 'backend' should not be marked as external")
	}

	t.Logf("Successfully reconstructed stack YAML (%d bytes)", len(yaml))
}

func TestReconstructStackCompose_RoundTrip(t *testing.T) {
	stackName := "demo"

	// Step 1: Reconstruct YAML from the running stack
	yamlBefore, err := docker.ReconstructStackCompose(stackName)
	if err != nil {
		t.Fatalf("Failed to reconstruct stack compose (before): %v", err)
	}

	// Step 2: Redeploy from reconstructed YAML
	if err := docker.DeployStack(stackName, yamlBefore); err != nil {
		t.Fatalf("Failed to redeploy stack from reconstructed YAML: %v", err)
	}

	// Step 3: Wait for services to converge
	deadline := time.Now().Add(60 * time.Second)
	for time.Now().Before(deadline) {
		_, err := docker.GetOrRefreshSnapshot()
		if err == nil {
			break
		}
		time.Sleep(2 * time.Second)
	}
	// Extra settle time for service convergence
	time.Sleep(5 * time.Second)

	// Step 4: Reconstruct again
	yamlAfter, err := docker.ReconstructStackCompose(stackName)
	if err != nil {
		t.Fatalf("Failed to reconstruct stack compose (after): %v", err)
	}

	// Step 5: Parse both as ComposeFile structs and compare
	var before, after docker.ComposeFile
	if err := yaml.Unmarshal([]byte(yamlBefore), &before); err != nil {
		t.Fatalf("Failed to parse yamlBefore: %v", err)
	}
	if err := yaml.Unmarshal([]byte(yamlAfter), &after); err != nil {
		t.Fatalf("Failed to parse yamlAfter: %v", err)
	}

	// Assert top-level volume keys are identical
	if len(before.Volumes) != len(after.Volumes) {
		t.Errorf("Volume count changed: before=%d, after=%d", len(before.Volumes), len(after.Volumes))
	}
	for k := range before.Volumes {
		if _, ok := after.Volumes[k]; !ok {
			t.Errorf("Volume %q lost after round-trip", k)
		}
	}

	// Assert top-level network keys are identical
	if len(before.Networks) != len(after.Networks) {
		t.Errorf("Network count changed: before=%d, after=%d", len(before.Networks), len(after.Networks))
	}
	for k := range before.Networks {
		if _, ok := after.Networks[k]; !ok {
			t.Errorf("Network %q lost after round-trip", k)
		}
	}

	// Assert service-level volume mounts survived
	for svcName, svcBefore := range before.Services {
		svcAfter, ok := after.Services[svcName]
		if !ok {
			t.Errorf("Service %q lost after round-trip", svcName)
			continue
		}
		if len(svcBefore.Volumes) != len(svcAfter.Volumes) {
			t.Errorf("Service %q volume mounts changed: before=%v, after=%v", svcName, svcBefore.Volumes, svcAfter.Volumes)
		}
	}

	t.Logf("Round-trip test passed: %d volumes, %d networks preserved", len(after.Volumes), len(after.Networks))
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
