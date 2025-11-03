//go:build integration

package service

import (
	"testing"
	"time"

	"swarmcli/docker"
)

// TestScaleWhoamiService verifies scaling demo_whoami up and down safely.
func TestScaleWhoamiService(t *testing.T) {
	t.Parallel()
	const serviceName = "demo_whoami"
	const timeout = 45 * time.Second

	snap, err := docker.RefreshSnapshot()
	if err != nil {
		t.Fatalf("failed to refresh snapshot: %v", err)
	}

	svc := snap.FindServiceByName(serviceName)
	if svc == nil {
		t.Skipf("service %s not found; skipping", serviceName)
	}
	if svc.Spec.Mode.Replicated == nil {
		t.Skipf("service %s not in replicated mode; skipping", serviceName)
	}

	original := *svc.Spec.Mode.Replicated.Replicas
	if original < 2 {
		t.Fatalf("expected demo_whoami to start with at least 2 replicas, got %d", original)
	}

	scaleTarget := original - 1
	t.Logf("Scaling service %s down from %d → %d replicas", serviceName, original, scaleTarget)
	if err := docker.ScaleServiceByName(serviceName, scaleTarget); err != nil {
		t.Fatalf("failed to scale down: %v", err)
	}

	waitUntil := time.Now().Add(timeout)
	for {
		snap2, _ := docker.RefreshSnapshot()
		svc2 := snap2.FindServiceByName(serviceName)
		if svc2 == nil {
			t.Fatalf("service %s disappeared", serviceName)
		}
		running := 0
		for _, task := range snap2.Tasks {
			if task.ServiceID == svc2.ID && task.Status.State == "running" {
				running++
			}
		}
		if *svc2.Spec.Mode.Replicated.Replicas == scaleTarget && running == int(scaleTarget) {
			break
		}
		if time.Now().After(waitUntil) {
			t.Fatalf("timeout waiting for scale down")
		}
		time.Sleep(1 * time.Second)
	}

	// Restore
	t.Logf("Restoring service %s back to %d replicas", serviceName, original)
	if err := docker.ScaleServiceByName(serviceName, original); err != nil {
		t.Fatalf("failed to restore original count: %v", err)
	}
}
