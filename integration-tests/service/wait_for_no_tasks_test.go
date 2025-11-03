package service

import (
	"context"
	"swarmcli/docker"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
)

func TestWaitForNoTasksTimeout(t *testing.T) {
	const svcName = "demo_whoami_single"

	c, err := docker.GetClient()
	if err != nil {
		t.Fatalf("docker client: %v", err)
	}
	defer c.Close()

	// Lookup the service ID
	services, err := c.ServiceList(context.Background(), types.ServiceListOptions{})
	if err != nil {
		t.Fatalf("listing services: %v", err)
	}

	var svcID string
	for _, s := range services {
		if s.Spec.Name == svcName {
			svcID = s.ID
			break
		}
	}

	if svcID == "" {
		t.Fatalf("service %s not found", svcName)
	}

	// Use a very short timeout to trigger failure
	err = docker.WaitForNoTasks(context.Background(), c, svcID, 0*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	t.Logf("Correctly timed out: %v", err)
}
