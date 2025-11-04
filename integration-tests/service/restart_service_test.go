//go:build integration

package service

import (
	"testing"
	"time"

	"swarmcli/docker"
)

// TestRestartWhoamiSingleService_Idiomatic verifies that RestartServiceIdiomatic
// properly forces a new task for a single-replica service using the snapshot model.
func TestRestartWhoamiSingleService(t *testing.T) {
	t.Parallel()
	const serviceName = "demo_whoami_single"
	const timeout = 45 * time.Second

	snap, err := docker.RefreshSnapshot()
	if err != nil {
		t.Fatalf("failed to refresh snapshot: %v", err)
	}

	svc := snap.FindServiceByName(serviceName)
	if svc == nil {
		t.Fatalf("service %s not found", serviceName)
	}
	if svc.Spec.Mode.Replicated == nil {
		t.Fatalf("service %s not in replicated mode", serviceName)
	}
	if *svc.Spec.Mode.Replicated.Replicas != 1 {
		t.Fatalf("expected %s to have exactly 1 replica, got %d", serviceName, *svc.Spec.Mode.Replicated.Replicas)
	}

	// Get current running task ID
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

	t.Logf("Restarting service %s (old task ID: %s)", serviceName, oldTaskID)
	start := time.Now()
	if err := docker.RestartService(serviceName); err != nil {
		t.Fatalf("failed to restart service: %v", err)
	}

	waitUntil := time.Now().Add(timeout)
	var newTaskID string
	for {
		snap2, _ := docker.RefreshSnapshot()
		svc2 := snap2.FindServiceByName(serviceName)
		if svc2 == nil {
			t.Fatalf("service %s disappeared after restart", serviceName)
		}
		for _, task := range snap2.Tasks {
			if task.ServiceID == svc2.ID && task.Status.State == "running" {
				newTaskID = task.ID
				break
			}
		}
		if newTaskID != "" && newTaskID != oldTaskID {
			break // success
		}
		if time.Now().After(waitUntil) {
			t.Fatalf("timeout waiting for new running task after restart (old: %s, new: %s)", oldTaskID, newTaskID)
		}
		time.Sleep(1 * time.Second)
	}

	t.Logf("✅ Service %s restarted successfully (old task %s → new task %s) in %v",
		serviceName, oldTaskID, newTaskID, time.Since(start))
}

// TestRestartWhoamiMultiService_Idiomatic verifies rolling restart behavior
// for multi-replica services using snapshots.
func TestRestartWhoamiMultiService(t *testing.T) {
	t.Parallel()
	const serviceName = "demo_whoami"
	const timeout = 90 * time.Second

	snap, err := docker.RefreshSnapshot()
	if err != nil {
		t.Fatalf("failed to refresh snapshot: %v", err)
	}

	svc := snap.FindServiceByName(serviceName)
	if svc == nil {
		t.Fatalf("service %s not found", serviceName)
	}
	if svc.Spec.Mode.Replicated == nil {
		t.Fatalf("service %s not in replicated mode", serviceName)
	}

	replicas := *svc.Spec.Mode.Replicated.Replicas
	if replicas < 2 {
		t.Fatalf("expected at least 2 replicas, got %d", replicas)
	}

	oldTasks := map[string]bool{}
	for _, task := range snap.Tasks {
		if task.ServiceID == svc.ID && task.Status.State == "running" {
			oldTasks[task.ID] = true
		}
	}
	if len(oldTasks) != int(replicas) {
		t.Logf("⚠️  expected %d running tasks, got %d", replicas, len(oldTasks))
	}

	t.Logf("Restarting multi-replica service %s (%d replicas)", serviceName, replicas)
	start := time.Now()
	if err := docker.RestartService(serviceName); err != nil {
		t.Fatalf("failed to restart service: %v", err)
	}

	waitUntil := time.Now().Add(timeout)
	var changed bool
	for {
		snap2, _ := docker.RefreshSnapshot()
		svc2 := snap2.FindServiceByName(serviceName)
		if svc2 == nil {
			t.Fatalf("service %s disappeared after restart", serviceName)
		}
		running := 0
		newTasks := map[string]bool{}
		for _, task := range snap2.Tasks {
			if task.ServiceID == svc2.ID && task.Status.State == "running" {
				newTasks[task.ID] = true
				running++
			}
		}
		if running == int(replicas) {
			for id := range newTasks {
				if !oldTasks[id] {
					changed = true
					break
				}
			}
			if changed {
				break
			}
		}
		if time.Now().After(waitUntil) {
			t.Fatalf("timeout waiting for rolling update of %s", serviceName)
		}
		time.Sleep(2 * time.Second)
	}

	t.Logf("✅ Multi-replica service %s rolled out new tasks successfully in %v", serviceName, time.Since(start))
}

func TestRestartService_NotFound_ReturnsError(t *testing.T) {
	t.Parallel()
	const serviceName = "nonexistent_demo_service"

	err := docker.RestartService(serviceName)
	if err == nil {
		t.Fatalf("expected error when restarting nonexistent service %s, got nil", serviceName)
	}

	t.Logf("✅ Correctly returned error for nonexistent service: %v", err)
}
