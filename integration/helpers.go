//go:build integration
// +build integration

package integration

import (
	"context"
	"swarmcli/docker"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

// EnsureSwarmReady checks if Docker and Swarm are reachable and active.
func EnsureSwarmReady(t *testing.T) *client.Client {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	c, err := docker.GetClient()
	if err != nil {
		t.Fatalf("failed to create docker client: %v", err)
	}

	info, err := c.Info(ctx)
	if err != nil {
		t.Fatalf("docker not reachable: %v", err)
	}
	if !info.Swarm.ControlAvailable {
		t.Fatalf("swarm not active: %+v", info.Swarm)
	}

	t.Logf("✅ Connected to Swarm manager: %s (%s)", info.Name, info.Swarm.NodeID)
	return c
}

// WaitForStack ensures the named stack is deployed and has services running.
func WaitForStack(t *testing.T, c *client.Client, stackName string, timeout time.Duration) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		services, err := c.ServiceList(ctx, types.ServiceListOptions{})
		if err != nil {
			t.Fatalf("failed to list services: %v", err)
		}

		count := 0
		for _, s := range services {
			if s.Spec.Labels["com.docker.stack.namespace"] == stackName {
				count++
			}
		}

		if count > 0 {
			t.Logf("✅ Stack %q ready with %d service(s)", stackName, count)
			return
		}

		select {
		case <-ctx.Done():
			t.Fatalf("timeout waiting for stack %q", stackName)
		case <-time.After(2 * time.Second):
		}
	}
}
