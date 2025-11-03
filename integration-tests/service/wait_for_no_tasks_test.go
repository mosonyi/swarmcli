package service

import (
	"context"
	"swarmcli/docker"
	"testing"
	"time"
)

func TestWaitForNoTasksTimeout(t *testing.T) {
	const svcName = "demo_whoami_single"

	c, err := docker.GetClient()
	if err != nil {
		t.Fatalf("docker client: %v", err)
	}
	defer c.Close()

	// Use a very short timeout to trigger failure
	err = docker.WaitForNoTasks(context.Background(), c, svcName, 1*time.Millisecond)
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
}
