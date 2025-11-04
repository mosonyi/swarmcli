package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
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

// RestartService restarts a replicated service by performing
// a `docker service update --force` equivalent. This triggers rolling restarts
// according to the service’s update configuration, regardless of replica count.
func RestartService(serviceName string) error {
	c, err := GetClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	defer c.Close()

	ctx := context.Background()

	// Find the service by name
	services, err := c.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		return fmt.Errorf("listing services: %w", err)
	}

	var svc *swarm.Service
	for i := range services {
		if services[i].Spec.Name == serviceName {
			svc = &services[i]
			break
		}
	}

	if svc == nil {
		return fmt.Errorf("service %s not found", serviceName)
	}

	// Ensure the service uses replicated mode (not global)
	if svc.Spec.Mode.Replicated == nil {
		return fmt.Errorf("service %s is not in replicated mode (global services are not supported)", serviceName)
	}

	// Increment the ForceUpdate counter — this is how Docker signals a restart
	svc.Spec.TaskTemplate.ForceUpdate++

	updateOpts := types.ServiceUpdateOptions{
		RegistryAuthFrom: types.RegistryAuthFromSpec,
	}

	resp, err := c.ServiceUpdate(ctx, svc.ID, svc.Version, svc.Spec, updateOpts)
	if err != nil {
		return fmt.Errorf("forcing service update for %s: %w", serviceName, err)
	}

	// Print any warnings returned by Docker
	for _, w := range resp.Warnings {
		fmt.Printf("⚠️  Warning during restart of %s: %s\n", serviceName, w)
	}

	fmt.Printf("🔁 Service %s restarted (replicas: %d)\n",
		serviceName, *svc.Spec.Mode.Replicated.Replicas)

	return nil
}

// RestartServiceAndWait restarts a service `RestartService`
// and waits until a new task is running again, or the context expires.
// It also verifies that the new task ID differs from the previous one.
func RestartServiceAndWait(ctx context.Context, serviceName string) error {
	c, err := GetClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	defer c.Close()

	// Step 1: Locate service and its current running task (if any)
	services, err := c.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		return fmt.Errorf("listing services: %w", err)
	}

	var svc *swarm.Service
	for i := range services {
		if services[i].Spec.Name == serviceName {
			svc = &services[i]
			break
		}
	}
	if svc == nil {
		return fmt.Errorf("service %s not found", serviceName)
	}

	oldTaskID := ""
	tasks, err := c.TaskList(ctx, types.TaskListOptions{})
	if err != nil {
		return fmt.Errorf("listing tasks: %w", err)
	}
	for _, t := range tasks {
		if t.ServiceID == svc.ID && t.Status.State == "running" {
			oldTaskID = t.ID
			break
		}
	}

	// Step 2: Restart safely
	if err := RestartService(serviceName); err != nil {
		return err
	}

	// Step 3: Wait for a new running task (different ID)
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for service %s: %w", serviceName, ctx.Err())

		default:
			tasks, err := c.TaskList(ctx, types.TaskListOptions{})
			if err != nil {
				return fmt.Errorf("listing tasks: %w", err)
			}

			for _, t := range tasks {
				if t.ServiceID == svc.ID && t.Status.State == "running" {
					if t.ID != oldTaskID {
						return nil // New task is running — restart complete
					}
				}
			}

			time.Sleep(500 * time.Millisecond)
		}
	}
}
