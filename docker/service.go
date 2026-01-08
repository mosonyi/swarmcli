package docker

import (
	"context"
	"fmt"
	"sort"
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
	services, err := c.ServiceList(ctx, swarm.ServiceListOptions{})
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
	resp, err := c.ServiceUpdate(ctx, svc.ID, svc.Version, svc.Spec, swarm.ServiceUpdateOptions{
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
	svc.Spec.TaskTemplate.ForceUpdate++
	if err := updateService(ctx, c, svc); err != nil {
		return fmt.Errorf("forcing service update for %s: %w", svc.Spec.Name, err)
	}

	// Log with mode-specific info
	if svc.Spec.Mode.Replicated != nil {
		l().Infof("🔁 Service %s restarted (replicas: %d)\n",
			svc.Spec.Name, *svc.Spec.Mode.Replicated.Replicas)
	} else {
		l().Infof("🔁 Service %s restarted (global mode)\n", svc.Spec.Name)
	}
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
	defer closeCli(c)

	ctx := context.Background()

	svc, _, err := c.ServiceInspectWithRaw(ctx, serviceID, swarm.ServiceInspectOptions{})
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
	defer closeCli(c)

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
	defer closeCli(c)

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
	defer closeCli(cli)

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
	tasks, err := cli.TaskList(ctx, swarm.TaskListOptions{})
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

		tasks, err := cli.TaskList(ctx, swarm.TaskListOptions{})
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

// Service Loader Helpers

type ServiceEntry struct {
	StackName      string
	ServiceName    string
	ServiceID      string
	ReplicasOnNode int
	ReplicasTotal  int
	Status         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func LoadNodeServices(nodeID string) []ServiceEntry {
	snap, err := GetOrRefreshSnapshot()
	if err != nil {
		l().Infof("failed to get snapshot:", err)
		return nil
	}

	var entries []ServiceEntry

	for _, svc := range snap.Services {
		stack, desired := getServiceStackAndDesired(svc, snap)

		// Count tasks running on this node
		onNode := countTasksForNode(svc.ID, nodeID, snap)

		// Skip if no tasks on this node
		if onNode == 0 {
			continue
		}

		entries = append(entries, ServiceEntry{
			StackName:      stack,
			ServiceName:    svc.Spec.Name,
			ServiceID:      svc.ID,
			ReplicasOnNode: onNode,
			ReplicasTotal:  desired,
			Status:         getServiceStatus(svc),
			CreatedAt:      svc.CreatedAt,
			UpdatedAt:      svc.UpdatedAt,
		})
	}

	sortEntries(entries)
	return entries
}

func LoadStackServices(stackName string) []ServiceEntry {
	snap, err := GetOrRefreshSnapshot()
	if err != nil {
		l().Infof("failed to get snapshot:", err)
		return nil
	}

	var entries []ServiceEntry

	for _, svc := range snap.Services {
		stack, desired := getServiceStackAndDesired(svc, snap)
		if stack != stackName {
			continue
		}

		// Count tasks on all nodes
		onNode := countTasksForNode(svc.ID, "", snap)

		entries = append(entries, ServiceEntry{
			StackName:      stack,
			ServiceName:    svc.Spec.Name,
			ServiceID:      svc.ID,
			ReplicasOnNode: onNode,
			ReplicasTotal:  desired,
			Status:         getServiceStatus(svc),
			CreatedAt:      svc.CreatedAt,
			UpdatedAt:      svc.UpdatedAt,
		})
	}

	sortEntries(entries)
	return entries
}

// --- Helpers ---

// getServiceStackAndDesired returns the stack name and desired replicas for a service
func getServiceStackAndDesired(svc swarm.Service, snap *SwarmSnapshot) (stack string, desired int) {
	stack = svc.Spec.Labels["com.docker.stack.namespace"]
	if stack == "" {
		stack = "-"
	}

	if svc.Spec.Mode.Replicated != nil && svc.Spec.Mode.Replicated.Replicas != nil {
		desired = int(*svc.Spec.Mode.Replicated.Replicas)
	} else if svc.Spec.Mode.Global != nil {
		desired = len(snap.Nodes)
	} else {
		// One-off task
		desired = 1
	}

	return
}

// countTasksForNode counts tasks for a service; if nodeID == "", counts across all nodes
func countTasksForNode(serviceID, nodeID string, snap *SwarmSnapshot) int {
	count := 0
	for _, t := range snap.Tasks {
		if t.ServiceID != serviceID {
			continue
		}
		if t.DesiredState != swarm.TaskStateRunning {
			continue
		}
		if nodeID == "" || t.NodeID == nodeID {
			count++
		}
	}
	return count
}

// sortEntries sorts entries by stack name then service name
func sortEntries(entries []ServiceEntry) {
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].StackName == entries[j].StackName {
			return entries[i].ServiceName < entries[j].ServiceName
		}
		return entries[i].StackName < entries[j].StackName
	})
}

// getServiceStatus returns a human-readable status string for a service
func getServiceStatus(svc swarm.Service) string {
	if svc.UpdateStatus != nil {
		switch svc.UpdateStatus.State {
		case swarm.UpdateStateUpdating:
			return "updating"
		case swarm.UpdateStatePaused:
			return "paused"
		case swarm.UpdateStateCompleted:
			return "updated"
		case swarm.UpdateStateRollbackStarted:
			return "rolling back"
		case swarm.UpdateStateRollbackPaused:
			return "rollback paused"
		case swarm.UpdateStateRollbackCompleted:
			return "rolled back"
		default:
			return string(svc.UpdateStatus.State)
		}
	}
	return "active"
}
