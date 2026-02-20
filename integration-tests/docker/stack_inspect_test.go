// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

//go:build integration
// +build integration

package docker_test

import (
	"encoding/json"
	"strings"
	"swarmcli/docker"
	"testing"
)

func TestInspectStack(t *testing.T) {
	// This test assumes that the demo stack is already deployed
	// (done by test-setup/testenv.sh deploy)
	stackName := "demo"

	jsonOutput, err := docker.GetStackInspection(stackName)
	if err != nil {
		t.Fatalf("Failed to inspect stack: %v", err)
	}

	if jsonOutput == "" {
		t.Fatal("Inspection output is empty")
	}

	// Verify it's valid JSON
	var inspection docker.StackInspection
	if err := json.Unmarshal([]byte(jsonOutput), &inspection); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify basic fields
	if inspection.Name != stackName {
		t.Errorf("Expected stack name %q, got %q", stackName, inspection.Name)
	}

	if inspection.ServiceCount == 0 {
		t.Error("Expected non-zero service count")
	}

	if len(inspection.Services) == 0 {
		t.Error("Expected services to be non-empty")
	}

	// Check that services have expected fields
	for _, svc := range inspection.Services {
		if svc.Name == "" {
			t.Error("Service missing name")
		}
		if svc.ID == "" {
			t.Error("Service missing ID")
		}
		if svc.Image == "" {
			t.Error("Service missing image")
		}
	}

	t.Logf("Successfully inspected stack: %d services, %d tasks", inspection.ServiceCount, inspection.TaskCount)
}

func TestInspectStack_NonExistent(t *testing.T) {
	stackName := "non-existent-stack-xyz123"

	_, err := docker.GetStackInspection(stackName)
	if err == nil {
		t.Fatal("Expected error for non-existent stack, got nil")
	}

	if !strings.Contains(err.Error(), "no services found") {
		t.Errorf("Expected 'no services found' error, got: %v", err)
	}
}
