package docker

import (
	"context"
	"fmt"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
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

// findServiceByName returns the service struct for a given name.
func findServiceByName(ctx context.Context, c *client.Client, serviceName string) (*swarm.Service, error) {
	services, err := c.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}

	for i := range services {
		if services[i].Spec.Name == serviceName {
			return &services[i], nil
		}
	}

	return nil, fmt.Errorf("service %s not found", serviceName)
}

// updateService safely applies a ServiceUpdate and logs any warnings.
func updateService(ctx context.Context, c *client.Client, svc *swarm.Service) error {
	resp, err := c.ServiceUpdate(ctx, svc.ID, svc.Version, svc.Spec, types.ServiceUpdateOptions{
		RegistryAuthFrom: types.RegistryAuthFromSpec,
	})
	if err != nil {
		return fmt.Errorf("updating service %s: %w", svc.Spec.Name, err)
	}

	for _, w := range resp.Warnings {
		fmt.Printf("⚠️  Warning for service %s: %s\n", svc.Spec.Name, w)
	}

	return nil
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

	if svc.Spec.Mode.Replicated == nil {
		return fmt.Errorf("service %s is not in replicated mode", svc.Spec.Name)
	}

	current := *svc.Spec.Mode.Replicated.Replicas
	if current == replicas {
		return nil // nothing to change
	}

	svc.Spec.Mode.Replicated.Replicas = &replicas
	if err := updateService(ctx, c, &svc); err != nil {
		return fmt.Errorf("updating service %s replicas from %d to %d: %w",
			svc.Spec.Name, current, replicas, err)
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

	svc, err := findServiceByName(ctx, c, serviceName)
	if err != nil {
		return err
	}

	svc.Spec.Mode.Replicated.Replicas = &replicas
	return updateService(ctx, c, svc)
}

// RestartService restarts a replicated service by performing
// a `docker service update --force` equivalent.
func RestartService(serviceName string) error {
	c, err := GetClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	defer c.Close()

	ctx := context.Background()

	svc, err := findServiceByName(ctx, c, serviceName)
	if err != nil {
		return err
	}

	if svc.Spec.Mode.Replicated == nil {
		return fmt.Errorf("service %s is not in replicated mode (global not supported)", serviceName)
	}

	svc.Spec.TaskTemplate.ForceUpdate++
	if err := updateService(ctx, c, svc); err != nil {
		return fmt.Errorf("forcing service update for %s: %w", serviceName, err)
	}

	fmt.Printf("🔁 Service %s restarted (replicas: %d)\n",
		serviceName, *svc.Spec.Mode.Replicated.Replicas)
	return nil
}

// RestartServiceAndWait restarts a service safely (via --force)
// and waits until all tasks have been replaced with new ones.
func RestartServiceAndWait(ctx context.Context, serviceName string) error {
	c, err := GetClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	defer c.Close()

	svc, err := findServiceByName(ctx, c, serviceName)
	if err != nil {
		return err
	}
	if svc.Spec.Mode.Replicated == nil {
		return fmt.Errorf("service %s is not in replicated mode", serviceName)
	}

	replicas := *svc.Spec.Mode.Replicated.Replicas

	// Snapshot old running tasks
	oldTasks := map[string]bool{}
	tasks, err := c.TaskList(ctx, types.TaskListOptions{})
	if err != nil {
		return fmt.Errorf("listing tasks: %w", err)
	}
	for _, t := range tasks {
		if t.ServiceID == svc.ID && t.Status.State == "running" {
			oldTasks[t.ID] = true
		}
	}

	// Trigger restart
	if err := RestartService(serviceName); err != nil {
		return err
	}

	// Wait until all tasks replaced
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for service %s: %w", serviceName, ctx.Err())

		default:
			tasks, err := c.TaskList(ctx, types.TaskListOptions{})
			if err != nil {
				return fmt.Errorf("listing tasks: %w", err)
			}
			running := 0
			newCount := 0
			for _, t := range tasks {
				if t.ServiceID == svc.ID && t.Status.State == "running" {
					running++
					if !oldTasks[t.ID] {
						newCount++
					}
				}
			}

			if running == int(replicas) && newCount == running {
				return nil // all tasks replaced
			}

			time.Sleep(1 * time.Second)
		}
	}
}
