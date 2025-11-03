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
