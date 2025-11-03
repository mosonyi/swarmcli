package service

import (
	"testing"

	"swarmcli/docker"
)

// TestRestartServiceSafely ensures a single-replica service can be safely restarted
// without overlap, i.e. scaled down to 0 and up to 1 again.
func TestRestartServiceSafely(t *testing.T) {
	err := docker.RestartServiceSafely("demo_whoami_single")
	if err != nil {
		t.Fatalf("failed to safely restart service: %v", err)
	}
}

func TestRestartServiceSafelySingleReplica(t *testing.T) {
	const svcName = "demo_whoami_single"

	err := docker.RestartServiceSafely(svcName)
	if err != nil {
		t.Fatalf("failed to safely restart service: %v", err)
	}
}

func TestRestartServiceSafelyNonexistent(t *testing.T) {
	const svcName = "nonexistent_service"

	err := docker.RestartServiceSafely(svcName)
	if err == nil {
		t.Fatal("expected error for non-existent service, got nil")
	}
}
