// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"encoding/json"
	"fmt"
	"sort"
	"time"
)

// StackDescription contains detailed information about a stack
type StackDescription struct {
	Name         string           `json:"name"`
	Services     []ServiceSummary `json:"services"`
	Networks     []string         `json:"networks,omitempty"`
	Secrets      []string         `json:"secrets,omitempty"`
	Configs      []string         `json:"configs,omitempty"`
	ServiceCount int              `json:"service_count"`
	TaskCount    int              `json:"task_count"`
	CreatedAt    time.Time        `json:"created_at,omitempty"`
	UpdatedAt    time.Time        `json:"updated_at,omitempty"`
}

// ServiceSummary contains summary information about a service
type ServiceSummary struct {
	Name      string            `json:"name"`
	ID        string            `json:"id"`
	Image     string            `json:"image"`
	Mode      string            `json:"mode"`
	Replicas  string            `json:"replicas"`
	Ports     []string          `json:"ports,omitempty"`
	Labels    map[string]string `json:"labels,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	UpdatedAt time.Time         `json:"updated_at"`
}

// DescribeStack returns detailed information about a stack in JSON format
func DescribeStack(stackName string) (string, error) {
	snap, err := GetOrRefreshSnapshot()
	if err != nil {
		return "", fmt.Errorf("failed to get snapshot: %w", err)
	}

	desc := StackDescription{
		Name:     stackName,
		Services: []ServiceSummary{},
	}

	// Collect unique networks, secrets, and configs
	networksMap := make(map[string]bool)
	secretsMap := make(map[string]bool)
	configsMap := make(map[string]bool)

	var earliestCreated, latestUpdated time.Time

	// Find all services in this stack
	for _, svc := range snap.Services {
		stack := svc.Spec.Labels["com.docker.stack.namespace"]
		if stack != stackName {
			continue
		}

		// Track earliest/latest timestamps
		if earliestCreated.IsZero() || svc.CreatedAt.Before(earliestCreated) {
			earliestCreated = svc.CreatedAt
		}
		if latestUpdated.IsZero() || svc.UpdatedAt.After(latestUpdated) {
			latestUpdated = svc.UpdatedAt
		}

		// Count tasks for this service
		taskCount := 0
		for _, t := range snap.Tasks {
			if t.ServiceID == svc.ID {
				taskCount++
			}
		}
		desc.TaskCount += taskCount

		// Get service mode and replicas
		mode := "replicated"
		replicas := "?"
		if svc.Spec.Mode.Replicated != nil && svc.Spec.Mode.Replicated.Replicas != nil {
			replicas = fmt.Sprintf("%d/%d", taskCount, *svc.Spec.Mode.Replicated.Replicas)
		} else if svc.Spec.Mode.Global != nil {
			mode = "global"
			replicas = fmt.Sprintf("%d (global)", taskCount)
		}

		// Get image
		image := ""
		if svc.Spec.TaskTemplate.ContainerSpec != nil {
			image = svc.Spec.TaskTemplate.ContainerSpec.Image
		}

		// Get ports
		var ports []string
		if svc.Endpoint.Ports != nil {
			for _, p := range svc.Endpoint.Ports {
				proto := p.Protocol
				if proto == "" {
					proto = "tcp"
				}
				if p.PublishedPort != 0 {
					ports = append(ports, fmt.Sprintf("%d:%d/%s", p.PublishedPort, p.TargetPort, proto))
				} else {
					ports = append(ports, fmt.Sprintf("%d/%s", p.TargetPort, proto))
				}
			}
		}

		// Collect networks
		for _, net := range svc.Spec.TaskTemplate.Networks {
			if net.Target != "" {
				networksMap[net.Target] = true
			}
		}

		// Collect secrets
		if svc.Spec.TaskTemplate.ContainerSpec != nil {
			for _, s := range svc.Spec.TaskTemplate.ContainerSpec.Secrets {
				if s.SecretName != "" {
					secretsMap[s.SecretName] = true
				}
			}

			// Collect configs
			for _, c := range svc.Spec.TaskTemplate.ContainerSpec.Configs {
				if c.ConfigName != "" {
					configsMap[c.ConfigName] = true
				}
			}
		}

		// Add service summary
		desc.Services = append(desc.Services, ServiceSummary{
			Name:      svc.Spec.Name,
			ID:        svc.ID,
			Image:     image,
			Mode:      mode,
			Replicas:  replicas,
			Ports:     ports,
			Labels:    svc.Spec.Labels,
			CreatedAt: svc.CreatedAt,
			UpdatedAt: svc.UpdatedAt,
		})
	}

	if len(desc.Services) == 0 {
		return "", fmt.Errorf("no services found in stack %q", stackName)
	}

	desc.ServiceCount = len(desc.Services)
	desc.CreatedAt = earliestCreated
	desc.UpdatedAt = latestUpdated

	// Convert maps to sorted slices
	for net := range networksMap {
		desc.Networks = append(desc.Networks, net)
	}
	sort.Strings(desc.Networks)

	for sec := range secretsMap {
		desc.Secrets = append(desc.Secrets, sec)
	}
	sort.Strings(desc.Secrets)

	for cfg := range configsMap {
		desc.Configs = append(desc.Configs, cfg)
	}
	sort.Strings(desc.Configs)

	// Sort services by name
	sort.Slice(desc.Services, func(i, j int) bool {
		return desc.Services[i].Name < desc.Services[j].Name
	})

	// Marshal to JSON with indentation
	jsonBytes, err := json.MarshalIndent(desc, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal stack description: %w", err)
	}

	return string(jsonBytes), nil
}
