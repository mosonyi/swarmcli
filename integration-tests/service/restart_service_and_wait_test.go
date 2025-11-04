//go:build integration

package service

import (
	"context"
	"testing"
	"time"

	"swarmcli/docker"
)

// TestRestartServiceAndWait_Success ensures RestartServiceAndWait blocks until
// a new running task appears for a single-replica service.
func TestRestartServiceAndWait_Success(t *testing.T) {
	const serviceName = "demo_whoami_single"

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	// Refresh snapshot to get the current running task
	snap, err := docker.RefreshSnapshot()
	if err != nil {
		t.Fatalf("failed to refresh snapshot: %v", err)
	}
	svc := snap.FindServiceByName(serviceName)
	if svc == nil {
		t.Fatalf("service %s not found", serviceName)
	}

	var oldTaskID string
	for _, task := range snap.Tasks {
		if task.ServiceID == svc.ID && task.Status.State == "running" {
			oldTaskID = task.ID
			break
		}
	}
	if oldTaskID == "" {
		t.Fatalf("no running task found for %s before restart", serviceName)
	}

	t.Logf("Restarting service %s and waiting for new task...", serviceName)

	start := time.Now()
	if err := docker.RestartServiceAndWait(ctx, serviceName); err != nil {
		t.Fatalf("RestartServiceAndWait failed: %v", err)
	}
	elapsed := time.Since(start)

	// Verify a new task has replaced the old one
	snap2, err := docker.RefreshSnapshot()
	if err != nil {
		t.Fatalf("failed to refresh snapshot after restart: %v", err)
	}
	var newTaskID string
	for _, task := range snap2.Tasks {
		if task.ServiceID == svc.ID && task.Status.State == "running" {
			newTaskID = task.ID
			break
		}
	}
	if newTaskID == "" {
		t.Fatalf("no running task found after restart")
	}
	if newTaskID == oldTaskID {
		t.Fatalf("expected a new running task, but got same task ID %s", oldTaskID)
	}

	t.Logf("✅ RestartServiceAndWait succeeded: old task %s → new task %s in %v",
		oldTaskID, newTaskID, elapsed)
}

// TestRestartServiceAndWait_Timeout verifies that RestartServiceAndWait
// returns a context timeout error when the deadline is exceeded.
func TestRestartServiceAndWait_Timeout(t *testing.T) {
	const serviceName = "demo_whoami_single"

	// Intentionally use a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	start := time.Now()
	err := docker.RestartServiceAndWait(ctx, serviceName)
	elapsed := time.Since(start)

	if err == nil {
		t.Fatalf("expected timeout error, got nil")
	}
	t.Logf("✅ RestartServiceAndWait correctly timed out after %v: %v", elapsed, err)
}

// TestRestartServiceAndWait_ServiceNotFound checks behaviour for a missing service.
func TestRestartServiceAndWait_ServiceNotFound(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err := docker.RestartServiceAndWait(ctx, "nonexistent_demo_service")
	if err == nil {
		t.Fatalf("expected error when restarting nonexistent service, got nil")
	}
	t.Logf("✅ Correctly returned error for nonexistent service: %v", err)
}
