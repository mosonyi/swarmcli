//go:build integration

package tests

import (
	"os"
	"testing"

	"swarmcli/docker"
)

func TestStackEntriesFromSwarm(t *testing.T) {
	dockerHost := os.Getenv("DOCKER_HOST")
	if dockerHost == "" {
		t.Skip("DOCKER_HOST not set; skipping integration test")
	}

	snap, err := docker.RefreshSnapshot()
	if err != nil {
		t.Fatalf("failed to refresh snapshot: %v", err)
	}

	stacks := snap.ToStackEntries()
	if len(stacks) == 0 {
		t.Fatalf("expected at least one stack, got 0")
	}

	var found bool
	for _, s := range stacks {
		if s.Name == "demo" {
			found = true
			if s.ServiceCount < 1 {
				t.Errorf("expected demo stack to have at least one service, got %d", s.ServiceCount)
			}
		}
	}

	if !found {
		t.Fatalf("demo stack not found in snapshot")
	}
}
