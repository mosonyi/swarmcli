package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type StackService struct {
	NodeID         string
	StackName      string
	ServiceName    string
	ServiceID      string
	ReplicasOnNode int
	ReplicasTotal  int
}

// ScaleService updates the replica count of a service by ID.
func ScaleService(serviceID string, replicas uint64) error {
	c, err := GetClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	defer c.Close()

	ctx := context.Background()

	svc, _, err := c.ServiceInspectWithRaw(ctx, serviceID, types.ServiceInspectOptions{})
	if err != nil {
		return fmt.Errorf("inspect service %s: %w", serviceID, err)
	}

	// Only replicated services can be scaled
	if svc.Spec.Mode.Replicated == nil {
		return fmt.Errorf("service %s is not in replicated mode", svc.Spec.Name)
	}

	current := *svc.Spec.Mode.Replicated.Replicas
	if current == replicas {
		// Nothing to do
		return nil
	}

	svc.Spec.Mode.Replicated.Replicas = &replicas

	// Apply the update
	resp, err := c.ServiceUpdate(ctx, svc.ID, svc.Version, svc.Spec, types.ServiceUpdateOptions{})
	if err != nil {
		return fmt.Errorf("updating service %s replicas from %d to %d: %w", svc.Spec.Name, current, replicas, err)
	}

	if len(resp.Warnings) > 0 {
		for _, w := range resp.Warnings {
			fmt.Printf("⚠️  Warning scaling service %s: %s\n", svc.Spec.Name, w)
		}
	}

	return nil
}

// ScaleServiceByName looks up a service by name and scales it.
func ScaleServiceByName(serviceName string, replicas uint64) error {
	c, err := GetClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	defer c.Close()

	ctx := context.Background()

	services, err := c.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		return fmt.Errorf("listing services: %w", err)
	}

	var svcID string
	for _, svc := range services {
		if svc.Spec.Name == serviceName {
			svcID = svc.ID
			break
		}
	}

	if svcID == "" {
		return fmt.Errorf("service %s not found", serviceName)
	}

	return ScaleService(svcID, replicas)
}

// RestartServiceSafely scales a single-replica service down to 0 and back up to 1.
// Returns an error if the service has 0 or >1 replicas to prevent unsafe operations.
func RestartServiceSafely(serviceName string) error {
	c, err := GetClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	defer c.Close()

	ctx := context.Background()

	// Find the service and current replicas
	services, err := c.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		return fmt.Errorf("listing services: %w", err)
	}

	var svcID, svcName string
	var currentReplicas uint64

	for _, svc := range services {
		if svc.Spec.Name == serviceName {
			svcID = svc.ID
			svcName = svc.Spec.Name
			if svc.Spec.Mode.Replicated != nil && svc.Spec.Mode.Replicated.Replicas != nil {
				currentReplicas = *svc.Spec.Mode.Replicated.Replicas
			}
			break
		}
	}

	if svcID == "" {
		return fmt.Errorf("service %s not found", serviceName)
	}

	if currentReplicas != 1 {
		return fmt.Errorf("service %s has %d replicas; RestartServiceSafely requires exactly 1", svcName, currentReplicas)
	}

	// Step 1: Scale down to 0 replicas
	if err := ScaleService(svcID, 0); err != nil {
		return fmt.Errorf("scale down failed: %w", err)
	}

	// Step 2: Wait until all tasks are actually removed
	if err := WaitForNoTasks(ctx, c, svcID, 10*time.Second); err != nil {
		return fmt.Errorf("waiting for tasks to stop: %w", err)
	}

	// Step 3: Scale up again to 1 replica
	if err := ScaleService(svcID, 1); err != nil {
		return fmt.Errorf("scale up failed: %w", err)
	}

	return nil
}

// WaitForNoTasks waits until the given service has no running tasks left.
func WaitForNoTasks(ctx context.Context, c *client.Client, serviceID string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)

	for {
		tasks, err := c.TaskList(ctx, types.TaskListOptions{})
		if err != nil {
			return fmt.Errorf("listing tasks: %w", err)
		}

		active := 0
		for _, t := range tasks {
			if t.ServiceID == serviceID && t.Status.State != "shutdown" && t.Status.State != "complete" {
				active++
			}
		}

		if active == 0 {
			return nil
		}

		if time.Now().After(deadline) {
			return fmt.Errorf("timeout waiting for service %s to stop (%d still active)", serviceID, active)
		}

		time.Sleep(500 * time.Millisecond)
	}
}
