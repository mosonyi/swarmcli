//go:build integration
// +build integration

package integration

import (
	"testing"

	"swarmcli/docker"
)

func TestStacksViewListsDemoStack(t *testing.T) {
	_, err := docker.RefreshSnapshot()
	if err != nil {
		t.Fatalf("failed to refresh snapshot: %v", err)
	}

	snap := docker.GetSnapshot()
	if snap == nil {
		t.Fatal("expected non-nil snapshot")
	}

	found := false
	for _, s := range snap.ToStackEntries() {
		if s.Name == "demo" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find stack 'demo', got %+v", snap.ToStackEntries())
	}

	t.Log("✅ Stack 'demo' found in snapshot")
}
