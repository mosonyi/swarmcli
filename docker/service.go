package docker

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/swarm"
	"github.com/docker/docker/client"
)

// StackService is a lightweight representation of a Swarm service within a stack.
type StackService struct {
	NodeID         string
	StackName      string
	ServiceName    string
	ServiceID      string
	ReplicasOnNode int
	ReplicasTotal  int
}

//
// ─── Internal helpers ───────────────────────────────────────────────────────────
//

// findServiceByName returns a swarm.Service by name, or an error if not found.
func findServiceByName(ctx context.Context, c *client.Client, name string) (*swarm.Service, error) {
	services, err := c.ServiceList(ctx, types.ServiceListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing services: %w", err)
	}
	for i := range services {
		if services[i].Spec.Name == name {
			return &services[i], nil
		}
	}
	return nil, fmt.Errorf("service %s not found", name)
}

// updateService applies the given Service.Spec and logs any warnings.
func updateService(ctx context.Context, c *client.Client, svc *swarm.Service) error {
	resp, err := c.ServiceUpdate(ctx, svc.ID, svc.Version, svc.Spec, types.ServiceUpdateOptions{
		RegistryAuthFrom: types.RegistryAuthFromSpec,
	})
	if err != nil {
		return fmt.Errorf("updating service %s: %w", svc.Spec.Name, err)
	}
	for _, w := range resp.Warnings {
		l().Warnf("⚠️  Warning for service %s: %s\n", svc.Spec.Name, w)
	}
	return nil
}

// scaleServiceCommon performs the actual scaling given a service struct.
func scaleServiceCommon(ctx context.Context, c *client.Client, svc *swarm.Service, replicas uint64) error {
	if svc.Spec.Mode.Replicated == nil {
		return fmt.Errorf("service %s is not in replicated mode", svc.Spec.Name)
	}
	current := *svc.Spec.Mode.Replicated.Replicas
	if current == replicas {
		return nil // nothing to change
	}
	svc.Spec.Mode.Replicated.Replicas = &replicas
	return updateService(ctx, c, svc)
}

// restartServiceCommon increments ForceUpdate to trigger a rolling restart.
func restartServiceCommon(ctx context.Context, c *client.Client, svc *swarm.Service) error {
	if svc.Spec.Mode.Replicated == nil {
		return fmt.Errorf("service %s is not in replicated mode (global not supported)", svc.Spec.Name)
	}
	svc.Spec.TaskTemplate.ForceUpdate++
	if err := updateService(ctx, c, svc); err != nil {
		return fmt.Errorf("forcing service update for %s: %w", svc.Spec.Name, err)
	}
	l().Warnf("🔁 Service %s restarted (replicas: %d)\n",
		svc.Spec.Name, *svc.Spec.Mode.Replicated.Replicas)
	return nil
}

//
// ─── Public API ─────────────────────────────────────────────────────────────────
//

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
	return scaleServiceCommon(ctx, c, &svc, replicas)
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
	return scaleServiceCommon(ctx, c, svc, replicas)
}

// RestartService performs a rolling restart (like `docker service update --force`).
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
	return restartServiceCommon(ctx, c, svc)
}

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
	oldTasks := make(map[string]bool)
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
	if err := restartServiceCommon(ctx, c, svc); err != nil {
		return err
	}

	// Wait until all running tasks are replaced
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for service %s: %w", serviceName, ctx.Err())
		default:
			tasks, err := c.TaskList(ctx, types.TaskListOptions{})
			if err != nil {
				return fmt.Errorf("listing tasks: %w", err)
			}

			running, newCount := 0, 0
			for _, t := range tasks {
				if t.ServiceID == svc.ID && t.Status.State == "running" {
					running++
					if !oldTasks[t.ID] {
						newCount++
					}
				}
			}

			if running == int(replicas) && newCount == running {
				// Check for extra tasks
				var extra []string
				for _, t := range tasks {
					if t.ServiceID == svc.ID && t.Status.State == "running" && !oldTasks[t.ID] && newCount > int(replicas) {
						extra = append(extra, t.ID)
					}
				}
				if len(extra) > 0 {
					l().Warnf("[RestartServiceAndWait] Warning: extra running tasks detected for service %q: %v\n", serviceName, extra)
				}

				return nil // all tasks replaced
			}

			time.Sleep(time.Second)
		}
	}
}

type ProgressUpdate struct {
	Replaced int
	Total    int
}

func RestartServiceWithProgress(ctx context.Context, serviceName string, progressCh chan<- ProgressUpdate) error {
	cli, err := GetClient()
	if err != nil {
		return err
	}
	defer cli.Close()

	svc, err := findServiceByName(ctx, cli, serviceName)
	if err != nil {
		return err
	}

	if svc.Spec.Mode.Replicated == nil {
		return fmt.Errorf("service %s is not replicated", serviceName)
	}
	total := int(*svc.Spec.Mode.Replicated.Replicas)

	// Snapshot old tasks
	oldTasks := make(map[string]bool)
	tasks, _ := cli.TaskList(ctx, types.TaskListOptions{})
	for _, t := range tasks {
		if t.ServiceID == svc.ID && t.Status.State == "running" {
			oldTasks[t.ID] = true
		}
	}

	// Trigger restart
	if err := restartServiceCommon(ctx, cli, svc); err != nil {
		return err
	}

	// Wait and report progress
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			tasks, _ := cli.TaskList(ctx, types.TaskListOptions{})
			replaced := 0
			running := 0
			for _, t := range tasks {
				if t.ServiceID == svc.ID && t.Status.State == "running" {
					running++
					if !oldTasks[t.ID] {
						replaced++
					}
				}
			}

			log.Printf("Sending progress: %d/%d\n", replaced, total)
			if progressCh != nil {
				select {
				case progressCh <- ProgressUpdate{Replaced: replaced, Total: total}:
					log.Printf("[Docker] Successfully sent to channel")
				case <-ctx.Done():
					log.Printf("[Docker] Context done, cannot send")
				default:
					log.Printf("[Docker] Channel blocked, skipping send")
				}
			}

			if running == total && replaced == total {
				return nil
			}
			time.Sleep(time.Second)
		}
	}
}
