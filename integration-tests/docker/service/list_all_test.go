// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

//go:build integration

package service

import (
	"testing"

	"github.com/Eldara-Tech/swarmcli/v2/docker"
)

// TestLoadAllServicesListsEveryService verifies that LoadAllServices returns
// every service in the swarm across all stacks (the data backing :service /
// :svc), not just the services of a single stack.
func TestLoadAllServicesListsEveryService(t *testing.T) {
	if _, err := docker.RefreshSnapshot(); err != nil {
		t.Fatalf("failed to refresh snapshot: %v", err)
	}

	all := docker.LoadAllServices()
	if len(all) == 0 {
		t.Fatal("LoadAllServices returned no services; expected the demo stack services")
	}

	got := make(map[string]bool, len(all))
	for _, e := range all {
		got[e.ServiceName] = true
	}

	// Services deployed by test-setup/test-stack.yml under the "demo" stack.
	for _, name := range []string{
		"demo_whoami",
		"demo_whoami_single",
		"demo_default_net_probe",
		"demo_log_ticker",
	} {
		if !got[name] {
			t.Errorf("expected service %q in LoadAllServices result", name)
		}
	}

	// "All" must cover at least everything a single-stack scope returns.
	stackOnly := docker.LoadStackServices("demo")
	if len(all) < len(stackOnly) {
		t.Errorf("LoadAllServices returned %d services, fewer than the demo stack's %d",
			len(all), len(stackOnly))
	}
}
