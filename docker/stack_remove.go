// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/swarm"
)

// RemoveStack removes all services in a stack by stack name.
func RemoveStack(ctx context.Context, stackName string) error {
	c, err := GetClient()
	if err != nil {
		return fmt.Errorf("docker client: %w", err)
	}

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
		return fmt.Errorf("no services found in stack '%s'", stackName)
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

// GetStackNetworks returns the names of networks associated with a stack
func GetStackNetworks(ctx context.Context, stackName string) ([]string, error) {
	c, err := GetClient()
	if err != nil {
		return nil, fmt.Errorf("docker client: %w", err)
	}

	// List all networks
	networks, err := c.NetworkList(ctx, network.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing networks: %w", err)
	}

	// Filter networks by stack label
	var stackNetworks []string
	for _, net := range networks {
		if net.Labels != nil {
			if netStack, ok := net.Labels["com.docker.stack.namespace"]; ok && netStack == stackName {
				stackNetworks = append(stackNetworks, net.Name)
			}
		}
	}

	return stackNetworks, nil
}

// RemoveStackNetworks removes all networks associated with a stack
func RemoveStackNetworks(ctx context.Context, stackName string) error {
	networks, err := GetStackNetworks(ctx, stackName)
	if err != nil {
		return fmt.Errorf("getting stack networks: %w", err)
	}

	if len(networks) == 0 {
		l().Infof("No networks found for stack %s", stackName)
		return nil
	}
	var removedCount int
	for _, netName := range networks {
		if err := RemoveNetwork(ctx, netName); err != nil {
			l().Warnf("Failed to remove network %s: %v", netName, err)
		} else {
			l().Infof("Removed network: %s", netName)
			removedCount++
		}
	}

	l().Infof("🗑️  Removed %d/%d networks for stack %s", removedCount, len(networks), stackName)
	return nil
}
