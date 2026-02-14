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

func TestDescribeStack(t *testing.T) {
	// This test assumes that the test-stack is already deployed
	// (done by test-setup/testenv.sh deploy)
	stackName := "test-stack"

	jsonOutput, err := docker.DescribeStack(stackName)
	if err != nil {
		t.Fatalf("Failed to describe stack: %v", err)
	}

	if jsonOutput == "" {
		t.Fatal("Description output is empty")
	}

	// Verify it's valid JSON
	var desc docker.StackDescription
	if err := json.Unmarshal([]byte(jsonOutput), &desc); err != nil {
		t.Fatalf("Failed to parse JSON output: %v", err)
	}

	// Verify basic fields
	if desc.Name != stackName {
		t.Errorf("Expected stack name %q, got %q", stackName, desc.Name)
	}

	if desc.ServiceCount == 0 {
		t.Error("Expected non-zero service count")
	}

	if len(desc.Services) == 0 {
		t.Error("Expected services to be non-empty")
	}

	// Check that services have expected fields
	for _, svc := range desc.Services {
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

	t.Logf("Successfully described stack: %d services, %d tasks", desc.ServiceCount, desc.TaskCount)
}

func TestDescribeStack_NonExistent(t *testing.T) {
	stackName := "non-existent-stack-xyz123"

	_, err := docker.DescribeStack(stackName)
	if err == nil {
		t.Fatal("Expected error for non-existent stack, got nil")
	}

	if !strings.Contains(err.Error(), "no services found") {
		t.Errorf("Expected 'no services found' error, got: %v", err)
	}
}
