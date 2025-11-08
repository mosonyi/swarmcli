//go:build integration

package docker

import (
	"testing"

	"swarmcli/docker"
)

func TestStacksViewListsStacks(t *testing.T) {
	snap, err := docker.RefreshSnapshot()
	if err != nil {
		t.Fatalf("failed to refresh snapshot: %v", err)
	}

	stacks := snap.ToStackEntries()
	if len(stacks) == 0 {
		t.Fatal("expected at least one stack, got none")
	}

	found := false
	for _, s := range stacks {
		if s.Name == "demo" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected to find stack 'demo' in snapshot")
	}
}
