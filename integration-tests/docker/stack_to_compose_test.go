// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

//go:build integration
// +build integration

package docker_test

import (
	"strings"
	"swarmcli/docker"
	"testing"

	yamlPkg "gopkg.in/yaml.v3"
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

	// Parse YAML and verify backend network is not marked external
	var composed docker.ComposeFile
	if err := yamlPkg.Unmarshal([]byte(yaml), &composed); err != nil {
		t.Fatalf("Failed to parse reconstructed YAML: %v", err)
	}
	if netProps, ok := composed.Networks["backend"]; ok {
		if _, hasExternal := netProps["external"]; hasExternal {
			t.Error("Stack-managed network 'backend' should not be marked as external")
		}
	} else {
		t.Error("Parsed YAML missing 'backend' network key")
	}

	t.Logf("Successfully reconstructed stack YAML (%d bytes)", len(yaml))
}

func TestReconstructStackCompose_RoundTrip(t *testing.T) {
	// Round-trip: reconstruct → parse → verify volumes/networks survive structurally.
	// Full deploy round-trip is blocked by $$ escaping in commands (separate issue).
	stackName := "demo"

	yamlStr, err := docker.ReconstructStackCompose(stackName)
	if err != nil {
		t.Fatalf("Failed to reconstruct stack compose: %v", err)
	}

	var cf docker.ComposeFile
	if err := yamlPkg.Unmarshal([]byte(yamlStr), &cf); err != nil {
		t.Fatalf("Failed to parse reconstructed YAML: %v", err)
	}

	// Verify top-level volumes survived reconstruction
	if _, ok := cf.Volumes["whoami_data"]; !ok {
		t.Errorf("Volume 'whoami_data' missing from reconstructed compose, got keys: %v", mapKeys(cf.Volumes))
	}

	// Verify top-level networks survived reconstruction
	if _, ok := cf.Networks["backend"]; !ok {
		t.Errorf("Network 'backend' missing from reconstructed compose, got keys: %v", mapKeys(cf.Networks))
	}

	// Verify backend is not external
	if props, ok := cf.Networks["backend"]; ok {
		if _, hasExternal := props["external"]; hasExternal {
			t.Error("Stack-managed network 'backend' should not be marked as external after reconstruction")
		}
	}

	// Verify service-level volume mount
	if svc, ok := cf.Services["whoami"]; ok {
		found := false
		for _, v := range svc.Volumes {
			if strings.Contains(v, "whoami_data:") {
				found = true
			}
		}
		if !found {
			t.Errorf("Service 'whoami' missing volume mount for 'whoami_data', got: %v", svc.Volumes)
		}
	} else {
		t.Error("Service 'whoami' missing from reconstructed compose")
	}

	// Verify service-level network attachment
	if svc, ok := cf.Services["whoami"]; ok {
		nets, ok := svc.Networks.([]any)
		if !ok {
			t.Errorf("Service 'whoami' networks not a list, got %T", svc.Networks)
		} else {
			found := false
			for _, n := range nets {
				if n == "backend" {
					found = true
				}
			}
			if !found {
				t.Errorf("Service 'whoami' missing network 'backend', got: %v", nets)
			}
		}
	}

	t.Logf("Round-trip structural test passed: volumes=%v, networks=%v",
		mapKeys(cf.Volumes), mapKeys(cf.Networks))
}

func mapKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
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
