// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

//go:build integration
// +build integration

package docker_test

import (
	"encoding/json"
	"github.com/Eldara-Tech/swarmcli/v2/docker"
	"strings"
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

	// Verify volumes are present (stack prefix stripped: whoami_data, not demo_whoami_data)
	if len(inspection.Volumes) == 0 {
		t.Error("Expected non-empty volumes")
	}
	foundVolume := false
	for _, v := range inspection.Volumes {
		if v == "whoami_data" {
			foundVolume = true
		}
	}
	if !foundVolume {
		t.Errorf("Expected volume 'whoami_data' in volumes, got %v", inspection.Volumes)
	}

	// Verify networks are present with names (not hex IDs), stack prefix stripped
	if len(inspection.Networks) == 0 {
		t.Error("Expected non-empty networks")
	}
	foundNetwork := false
	for _, n := range inspection.Networks {
		// Network names should not be hex IDs (64-char hex strings)
		if len(n) == 64 && !strings.ContainsAny(n, "ghijklmnopqrstuvwxyz_-") {
			t.Errorf("Network appears to be an unresolved ID: %s", n)
		}
		if n == "backend" {
			foundNetwork = true
		}
	}
	if !foundNetwork {
		t.Errorf("Expected network 'backend' in networks, got %v", inspection.Networks)
	}

	// Verify secrets are present (top-level aggregated list)
	foundSecret := false
	for _, s := range inspection.Secrets {
		if s == "whoami_secret" {
			foundSecret = true
		}
	}
	if !foundSecret {
		t.Errorf("Expected secret 'whoami_secret' in secrets, got %v", inspection.Secrets)
	}

	// Verify configs are present (top-level aggregated list)
	foundConfig := false
	for _, c := range inspection.Configs {
		if c == "whoami_config" {
			foundConfig = true
		}
	}
	if !foundConfig {
		t.Errorf("Expected config 'whoami_config' in configs, got %v", inspection.Configs)
	}

	// Verify per-service secrets and configs
	for _, svc := range inspection.Services {
		switch {
		case strings.HasSuffix(svc.Name, "whoami") && !strings.Contains(svc.Name, "single"):
			// whoami service should have secrets and configs
			if len(svc.Secrets) == 0 {
				t.Errorf("Service %q: expected non-empty secrets", svc.Name)
			}
			if len(svc.Configs) == 0 {
				t.Errorf("Service %q: expected non-empty configs", svc.Name)
			}
		case strings.HasSuffix(svc.Name, "log_ticker"):
			// log_ticker has neither secrets nor configs
			if len(svc.Secrets) != 0 {
				t.Errorf("Service %q: expected no secrets, got %v", svc.Name, svc.Secrets)
			}
			if len(svc.Configs) != 0 {
				t.Errorf("Service %q: expected no configs, got %v", svc.Name, svc.Configs)
			}
		}

		// Issue #379: whoami_single declares a healthcheck; other services do not.
		if strings.HasSuffix(svc.Name, "whoami_single") {
			if svc.Healthcheck == nil {
				t.Errorf("Service %q: expected a healthcheck, got nil", svc.Name)
			} else {
				if svc.Healthcheck.Interval != "30s" {
					t.Errorf("Service %q: healthcheck interval = %q, want \"30s\"", svc.Name, svc.Healthcheck.Interval)
				}
				if svc.Healthcheck.Retries != 3 {
					t.Errorf("Service %q: healthcheck retries = %d, want 3", svc.Name, svc.Healthcheck.Retries)
				}
			}
		} else if svc.Healthcheck != nil {
			t.Errorf("Service %q: expected no healthcheck, got %+v", svc.Name, svc.Healthcheck)
		}

		// Issue #428: whoami_single declares a logging driver; others do not.
		if strings.HasSuffix(svc.Name, "whoami_single") {
			if svc.Logging == nil {
				t.Errorf("Service %q: expected a logging driver, got nil", svc.Name)
			} else {
				if svc.Logging.Driver != "json-file" {
					t.Errorf("Service %q: logging driver = %q, want \"json-file\"", svc.Name, svc.Logging.Driver)
				}
				if svc.Logging.Options["max-size"] != "10m" {
					t.Errorf("Service %q: logging option max-size = %q, want \"10m\"", svc.Name, svc.Logging.Options["max-size"])
				}
			}
		} else if svc.Logging != nil {
			t.Errorf("Service %q: expected no logging driver, got %+v", svc.Name, svc.Logging)
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
