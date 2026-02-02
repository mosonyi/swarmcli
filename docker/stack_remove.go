// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/swarm"
)

// RemoveStack removes all services in a stack by stack name.
func RemoveStack(stackName string) error {
	c, err := GetClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}
	defer closeCli(c)

	ctx := context.Background()

	// List all services in the stack
	services, err := c.ServiceList(ctx, swarm.ServiceListOptions{})
	if err != nil {
		return fmt.Errorf("listing services: %w", err)
	}

	// Filter services by stack label
	var stackServices []string
	for _, svc := range services {
		if svc.Spec.Labels != nil {
			if svcStack, ok := svc.Spec.Labels["com.docker.stack.namespace"]; ok && svcStack == stackName {
				stackServices = append(stackServices, svc.ID)
			}
		}
	}

	if len(stackServices) == 0 {
		return fmt.Errorf("no services found in stack %q", stackName)
	}

	// Remove all services in the stack
	for _, svcID := range stackServices {
		if err := c.ServiceRemove(ctx, svcID); err != nil {
			return fmt.Errorf("removing service %s: %w", svcID, err)
		}
	}

	l().Infof("🗑️  Stack %s removed (%d services)", stackName, len(stackServices))
	return nil
}
