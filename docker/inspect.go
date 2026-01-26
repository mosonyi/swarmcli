// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/docker/docker/api/types/swarm"
)

// InspectType enumerates supported resource types for inspect
type InspectType string

const (
	InspectNode      InspectType = "node"
	InspectService   InspectType = "service"
	InspectContainer InspectType = "container"
	InspectStack     InspectType = "stack"
)

// Inspect fetches and returns structured JSON for any Docker object.
func Inspect(ctx context.Context, t InspectType, id string) (string, error) {
	cli, err := GetClient()
	if err != nil {
		return "", fmt.Errorf("docker client: %w", err)
	}

	var obj any

	switch t {
	case InspectNode:
		node, _, err := cli.NodeInspectWithRaw(ctx, id)
		if err != nil {
			return "", fmt.Errorf("node inspect: %w", err)
		}
		obj = node

	case InspectService:
		svc, _, err := cli.ServiceInspectWithRaw(ctx, id, swarm.ServiceInspectOptions{})
		if err != nil {
			return "", fmt.Errorf("service inspect: %w", err)
		}
		obj = svc

	case InspectContainer:
		ctr, err := cli.ContainerInspect(ctx, id)
		if err != nil {
			return "", fmt.Errorf("container inspect: %w", err)
		}
		obj = ctr

	case InspectStack:
		// Fetch all services and filter by stack label
		services, err := cli.ServiceList(ctx, swarm.ServiceListOptions{})
		if err != nil {
			return "", fmt.Errorf("stack inspect: %w", err)
		}

		var stackServices []swarm.Service
		for _, s := range services {
			if s.Spec.Labels["com.docker.stack.namespace"] == id {
				stackServices = append(stackServices, s)
			}
		}

		if len(stackServices) == 0 {
			return "", fmt.Errorf("stack %q not found", id)
		}
		obj = stackServices

	default:
		return "", fmt.Errorf("unsupported inspect type: %s", t)
	}

	// Pretty-print JSON for TUI
	pretty, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal inspect result: %w", err)
	}

	return string(pretty), nil
}
