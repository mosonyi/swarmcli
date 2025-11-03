package service

import (
	"swarmcli/docker"
	"testing"
	"time"
)

func TestRestartServiceSafely_SingleReplica(t *testing.T) {
	serviceName := "demo_whoami_single"

	// Ensure service starts with 1 replica
	if err := docker.ScaleServiceByName(serviceName, 1); err != nil {
		t.Fatalf("failed to scale %s to 1: %v", serviceName, err)
	}

	start := time.Now()
	if err := docker.RestartServiceSafely(serviceName); err != nil {
		t.Fatalf("failed to restart service safely: %v", err)
	}
	elapsed := time.Since(start)

	t.Logf("Service %s restarted safely in %v", serviceName, elapsed)
}

func TestRestartServiceSafely_MultiReplica(t *testing.T) {
	serviceName := "demo_whoami_single"

	// Ensure service has >1 replica
	if err := docker.ScaleServiceByName(serviceName, 2); err != nil {
		t.Fatalf("failed to scale %s to 2: %v", serviceName, err)
	}

	err := docker.RestartServiceSafely(serviceName)
	if err == nil {
		t.Fatalf("expected error when restarting multi-replica service, got nil")
	}

	t.Logf("Correctly prevented restart for multi-replica service: %v", err)
}

func TestRestartServiceSafely_ServiceNotFound(t *testing.T) {
	serviceName := "non_existent_service"

	err := docker.RestartServiceSafely(serviceName)
	if err == nil {
		t.Fatalf("expected error when service does not exist, got nil")
	}

	t.Logf("Correctly returned error for missing service: %v", err)
}

func TestRestartServiceSafely_AlreadyZero(t *testing.T) {
	serviceName := "demo_whoami_single"

	// Scale down to 0 manually
	if err := docker.ScaleServiceByName(serviceName, 0); err != nil {
		t.Fatalf("failed to scale %s to 0: %v", serviceName, err)
	}

	// RestartServiceSafely should now return an error
	err := docker.RestartServiceSafely(serviceName)
	if err == nil {
		t.Fatalf("expected error when restarting service already at 0 replicas, got nil")
	}

	t.Logf("RestartServiceSafely correctly failed for service %s already at 0 replicas: %v", serviceName, err)
}
