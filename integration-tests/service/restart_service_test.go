package service

import (
	"testing"
	"time"

	"swarmcli/docker"
)

// Helper: fetch running task IDs for a given service.
func getRunningTaskIDs(t *testing.T, serviceName string) []string {
	t.Helper()
	tasks, err := docker.ListRunningTasks(serviceName)
	if err != nil {
		t.Fatalf("failed to list tasks for %s: %v", serviceName, err)
	}
	var ids []string
	for _, task := range tasks {
		ids = append(ids, task.ID)
	}
	if len(ids) == 0 {
		t.Fatalf("no running tasks found for %s", serviceName)
	}
	return ids
}

func TestRestartService_SingleReplica(t *testing.T) {
	serviceName := "demo_whoami_single"

	// Ensure service exists and runs one replica
	if err := docker.ScaleServiceByName(serviceName, 1); err != nil {
		t.Fatalf("failed to scale %s to 1: %v", serviceName, err)
	}
	time.Sleep(2 * time.Second)

	before := getRunningTaskIDs(t, serviceName)

	start := time.Now()
	if err := docker.RestartService(serviceName); err != nil {
		t.Fatalf("failed to restart service idiomatically: %v", err)
	}

	// Wait for new task to become ready
	time.Sleep(5 * time.Second)

	after := getRunningTaskIDs(t, serviceName)
	elapsed := time.Since(start)

	if len(before) != len(after) {
		t.Fatalf("expected same number of running tasks, before=%d after=%d", len(before), len(after))
	}

	if before[0] == after[0] {
		t.Fatalf("task ID did not change after restart — service may not have been updated")
	}

	t.Logf("✅ Service %s restarted idiomatically in %v (old task %s → new task %s)", serviceName, elapsed, before[0], after[0])
}

func TestRestartService_MultiReplica(t *testing.T) {
	serviceName := "demo_whoami"

	// Ensure multiple replicas exist
	if err := docker.ScaleServiceByName(serviceName, 3); err != nil {
		t.Fatalf("failed to scale %s to 3: %v", serviceName, err)
	}
	time.Sleep(3 * time.Second)

	before := getRunningTaskIDs(t, serviceName)
	if len(before) != 3 {
		t.Fatalf("expected 3 running tasks, got %d", len(before))
	}

	if err := docker.RestartService(serviceName); err != nil {
		t.Fatalf("expected rolling restart to succeed, got: %v", err)
	}

	time.Sleep(10 * time.Second)
	after := getRunningTaskIDs(t, serviceName)

	if len(after) != len(before) {
		t.Fatalf("expected same number of replicas after restart, got %d", len(after))
	}

	// Verify at least one task ID changed
	changed := false
	for _, idBefore := range before {
		found := false
		for _, idAfter := range after {
			if idAfter == idBefore {
				found = true
				break
			}
		}
		if !found {
			changed = true
			break
		}
	}

	if !changed {
		t.Fatalf("expected at least one task to be replaced during rolling update")
	}

	t.Logf("✅ Multi-replica service %s successfully rolled out new tasks", serviceName)
}

func TestRestartService_ServiceNotFound(t *testing.T) {
	serviceName := "non_existent_service"

	err := docker.RestartService(serviceName)
	if err == nil {
		t.Fatalf("expected error for nonexistent service %s, got nil", serviceName)
	}

	t.Logf("✅ Correctly returned error for nonexistent service: %v", err)
}

func TestRestartService_ZeroReplicas_NoError(t *testing.T) {
	serviceName := "demo_whoami_single"

	// Scale to 0 replicas
	if err := docker.ScaleServiceByName(serviceName, 0); err != nil {
		t.Fatalf("failed to scale %s to 0: %v", serviceName, err)
	}

	// Should not fail — should just no-op
	if err := docker.RestartService(serviceName); err != nil {
		t.Fatalf("expected no error when restarting 0-replica service, got: %v", err)
	}

	t.Logf("✅ Service %s correctly handled 0-replica restart as no-op", serviceName)
}
