// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"fmt"
	"strings"
	"time"

	"github.com/docker/docker/api/types/swarm"
)

// TaskEntry represents a task in a human-readable format
type TaskEntry struct {
	ID           string
	Name         string
	ServiceName  string
	Image        string
	NodeName     string
	DesiredState string
	CurrentState string
	Error        string
	Ports        string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// GetTasksForStack returns all tasks for services in the given stack
func GetTasksForStack(stackName string) ([]TaskEntry, error) {
	snap := GetSnapshot()
	if snap == nil {
		// No cached snapshot, try to refresh
		var err error
		snap, err = RefreshSnapshot()
		if err != nil {
			return nil, fmt.Errorf("failed to refresh snapshot: %w", err)
		}
	}

	var tasks []TaskEntry

	// Get all services for this stack
	stackServices := make(map[string]swarm.Service)
	for _, svc := range snap.Services {
		if svc.Spec.Labels["com.docker.stack.namespace"] == stackName {
			stackServices[svc.ID] = svc
		}
	}

	// Get nodes map for lookup
	nodesMap := make(map[string]string)
	for _, node := range snap.Nodes {
		nodesMap[node.ID] = node.Description.Hostname
	}

	// Filter tasks for this stack's services and sort by service name then created time
	for _, task := range snap.Tasks {
		if svc, ok := stackServices[task.ServiceID]; ok {
			nodeName := nodesMap[task.NodeID]
			if nodeName == "" {
				nodeName = task.NodeID[:12]
			}

			// Extract image name (without registry/tag details for cleaner display)
			imageParts := strings.Split(task.Spec.ContainerSpec.Image, "@")
			image := imageParts[0]
			if strings.Contains(image, ":") {
				image = strings.Split(image, ":")[0] + ":" + strings.Split(strings.Split(image, ":")[1], "@")[0]
			}

			// Format current state with timestamp
			currentState := string(task.Status.State)
			if !task.Status.Timestamp.IsZero() {
				duration := time.Since(task.Status.Timestamp)
				currentState = fmt.Sprintf("%s %s", currentState, formatTaskDuration(duration))
			}

			// Get error message if any
			errorMsg := ""
			if task.Status.Err != "" {
				errorMsg = task.Status.Err
				// Truncate long error messages
				if len(errorMsg) > 50 {
					errorMsg = errorMsg[:47] + "…"
				}
			}

			tasks = append(tasks, TaskEntry{
				ID:           task.ID[:12],
				Name:         fmt.Sprintf("%s.%d", svc.Spec.Name, task.Slot),
				ServiceName:  svc.Spec.Name,
				Image:        image,
				NodeName:     nodeName,
				DesiredState: string(task.DesiredState),
				CurrentState: currentState,
				Error:        errorMsg,
				Ports:        "", // Ports are typically on service level, not task level
				CreatedAt:    task.CreatedAt,
				UpdatedAt:    task.UpdatedAt,
			})
		}
	}

	// Sort tasks: by service name, then by created time (newest first for each service)
	sortTasksByServiceAndTime(tasks)

	return tasks, nil
}

func sortTasksByServiceAndTime(tasks []TaskEntry) {
	// Simple bubble sort for demonstration
	for i := 0; i < len(tasks); i++ {
		for j := i + 1; j < len(tasks); j++ {
			// First compare service names
			if tasks[i].ServiceName > tasks[j].ServiceName {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			} else if tasks[i].ServiceName == tasks[j].ServiceName {
				// Same service: sort by created time descending (newest first)
				if tasks[i].CreatedAt.Before(tasks[j].CreatedAt) {
					tasks[i], tasks[j] = tasks[j], tasks[i]
				}
			}
		}
	}
}

// GetTasksForService returns all tasks for a specific service ID from the cached snapshot.
func GetTasksForService(serviceID string) ([]TaskEntry, error) {
	snap := GetSnapshot()
	if snap == nil {
		return nil, fmt.Errorf("no snapshot available")
	}

	var tasks []TaskEntry

	// Build nodes map for hostname lookup
	nodesMap := make(map[string]string)
	for _, node := range snap.Nodes {
		nodesMap[node.ID] = node.Description.Hostname
	}

	// Filter tasks for this service and sort by created time
	for _, task := range snap.Tasks {
		if task.ServiceID == serviceID {
			nodeName := nodesMap[task.NodeID]
			if nodeName == "" {
				nodeName = task.NodeID[:12]
			}

			// Extract image name (without registry/tag details for cleaner display)
			imageParts := strings.Split(task.Spec.ContainerSpec.Image, "@")
			image := imageParts[0]
			if strings.Contains(image, ":") {
				image = strings.Split(image, ":")[0] + ":" + strings.Split(strings.Split(image, ":")[1], "@")[0]
			}

			// Format current state with timestamp
			currentState := string(task.Status.State)
			if !task.Status.Timestamp.IsZero() {
				duration := time.Since(task.Status.Timestamp)
				currentState = fmt.Sprintf("%s %s", currentState, formatTaskDuration(duration))
			}

			// Get error message if any
			errorMsg := ""
			if task.Status.Err != "" {
				errorMsg = task.Status.Err
				// Truncate long error messages
				if len(errorMsg) > 50 {
					errorMsg = errorMsg[:47] + "…"
				}
			}

			// Get service name from snapshot
			var serviceName string
			for _, svc := range snap.Services {
				if svc.ID == serviceID {
					serviceName = svc.Spec.Name
					break
				}
			}

			tasks = append(tasks, TaskEntry{
				ID:           task.ID[:12],
				Name:         fmt.Sprintf("%s.%d", serviceName, task.Slot),
				ServiceName:  serviceName,
				Image:        image,
				NodeName:     nodeName,
				DesiredState: string(task.DesiredState),
				CurrentState: currentState,
				Error:        errorMsg,
				Ports:        "", // Ports are typically on service level, not task level
				CreatedAt:    task.CreatedAt,
				UpdatedAt:    task.UpdatedAt,
			})
		}
	}

	// Sort tasks by created time (newest first)
	for i := 0; i < len(tasks); i++ {
		for j := i + 1; j < len(tasks); j++ {
			if tasks[i].CreatedAt.Before(tasks[j].CreatedAt) {
				tasks[i], tasks[j] = tasks[j], tasks[i]
			}
		}
	}

	return tasks, nil
}

func formatTaskDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%d seconds ago", int(d.Seconds()))
	} else if d < time.Hour {
		return fmt.Sprintf("%d minutes ago", int(d.Minutes()))
	} else if d < 24*time.Hour {
		return fmt.Sprintf("%d hours ago", int(d.Hours()))
	} else {
		days := int(d.Hours() / 24)
		if days < 7 {
			return fmt.Sprintf("%d days ago", days)
		} else if days < 30 {
			weeks := days / 7
			return fmt.Sprintf("%d weeks ago", weeks)
		} else {
			months := days / 30
			return fmt.Sprintf("%d months ago", months)
		}
	}
}
