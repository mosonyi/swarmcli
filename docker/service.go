package docker

import (
	"context"
	"fmt"
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
	l().Infof("🔁 Service %s restarted (replicas: %d)\n",
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

type ProgressUpdate struct {
	Replaced int
	Running  int
	Total    int
}

func RestartServiceAndWait(ctx context.Context, serviceName string) error {
	return restartServiceAndWaitInternal(ctx, serviceName, nil)
}

func RestartServiceWithProgress(ctx context.Context, serviceName string, progressCh chan<- ProgressUpdate) error {
	return restartServiceAndWaitInternal(ctx, serviceName, progressCh)
}

func restartServiceAndWaitInternal(ctx context.Context, serviceName string, progressCh chan<- ProgressUpdate) error {
	cli, err := GetClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	defer cli.Close()

	svc, err := findServiceByName(ctx, cli, serviceName)
	if err != nil {
		return err
	}
	if svc.Spec.Mode.Replicated == nil {
		return fmt.Errorf("service %s is not in replicated mode", serviceName)
	}

	total := int(*svc.Spec.Mode.Replicated.Replicas)
	l().Infof("🔁 Restarting service %s (replicas: %d)...", serviceName, total)

	// Snapshot old tasks
	oldTasks := map[string]swarm.Task{}
	tasks, err := cli.TaskList(ctx, types.TaskListOptions{})
	if err != nil {
		return fmt.Errorf("listing initial tasks: %w", err)
	}
	for _, t := range tasks {
		if t.ServiceID == svc.ID && t.DesiredState == swarm.TaskStateRunning {
			oldTasks[t.ID] = t
		}
	}
	l().Debugf("📦 Snapshot: %d old running tasks for %s", len(oldTasks), serviceName)

	// Trigger rolling restart
	if err := restartServiceCommon(ctx, cli, svc); err != nil {
		return fmt.Errorf("restart trigger: %w", err)
	}

	type slotState struct {
		oldTaskID string
		newTaskID string
	}
	slots := make(map[int]slotState)
	for _, t := range oldTasks {
		slots[t.Slot] = slotState{oldTaskID: t.ID}
	}

	var (
		lastProgress ProgressUpdate
		stableSince  time.Time
		lastActivity time.Time
	)
	lastActivity = time.Now()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("waiting for %s restart: %w", serviceName, ctx.Err())
		default:
		}

		tasks, err := cli.TaskList(ctx, types.TaskListOptions{})
		if err != nil {
			return fmt.Errorf("listing tasks: %w", err)
		}

		running := 0
		replaced := 0
		updating := 0

		for _, t := range tasks {
			if t.ServiceID != svc.ID || t.DesiredState != swarm.TaskStateRunning {
				continue
			}

			state := t.Status.State
			slot := t.Slot
			s := slots[slot]

			switch state {
			case swarm.TaskStateRunning:
				running++
				if s.oldTaskID != "" && t.ID != s.oldTaskID {
					s.newTaskID = t.ID
					replaced++
				}
				slots[slot] = s
			case swarm.TaskStatePreparing,
				swarm.TaskStateStarting,
				swarm.TaskStatePending,
				swarm.TaskStateAssigned:
				updating++
			}
		}

		currentProgress := ProgressUpdate{Replaced: replaced, Running: running, Total: total}
		if progressCh != nil && currentProgress != lastProgress {
			trySendProgress(progressCh, currentProgress)
			lastProgress = currentProgress
			lastActivity = time.Now()
			l().Debugf("[Docker] Progress update: %d/%d replaced, %d running", replaced, total, running)
		}

		// Determine stability
		allReplaced := true
		for _, s := range slots {
			if s.newTaskID == "" {
				allReplaced = false
				break
			}
		}

		if allReplaced && running >= total && updating == 0 {
			if stableSince.IsZero() {
				stableSince = time.Now()
			} else if time.Since(stableSince) > 3*time.Second {
				l().Infof("✅ Service %s stable: %d/%d new tasks running", serviceName, replaced, total)
				return nil
			}
		} else {
			stableSince = time.Time{}
		}

		// Adaptive polling — faster while changing, slower when idle
		sleep := 500 * time.Millisecond
		if time.Since(lastActivity) > 5*time.Second {
			sleep = 2 * time.Second
		}
		time.Sleep(sleep)
	}
}

func trySendProgress(ch chan<- ProgressUpdate, v ProgressUpdate) {
	select {
	case ch <- v:
	default:
		// drop silently; UI may be busy
	}
}
