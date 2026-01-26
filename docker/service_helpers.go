// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"fmt"
	"strings"

	"github.com/docker/docker/api/types/swarm"
)

// getServiceMode returns "replicated" or "global"
func getServiceMode(svc swarm.Service) string {
	if svc.Spec.Mode.Replicated != nil {
		return "replicated"
	} else if svc.Spec.Mode.Global != nil {
		return "global"
	}
	return "-"
}

// getServiceImage returns the image name without the digest
func getServiceImage(svc swarm.Service) string {
	if svc.Spec.TaskTemplate.ContainerSpec == nil {
		return "-"
	}
	image := svc.Spec.TaskTemplate.ContainerSpec.Image
	// Strip digest if present
	if idx := strings.Index(image, "@sha256:"); idx != -1 {
		image = image[:idx]
	}
	// Optionally truncate very long image names
	if len(image) > 50 {
		return image[:47] + "..."
	}
	return image
}

// getServicePorts returns a comma-separated list of published ports
func getServicePorts(svc swarm.Service) string {
	if len(svc.Endpoint.Ports) == 0 {
		return "-"
	}

	var ports []string
	for _, port := range svc.Endpoint.Ports {
		var portStr string
		if port.PublishedPort == 0 {
			continue
		}
		// Always show *: prefix for published ports (both ingress and host mode)
		portStr = fmt.Sprintf("*:%d", port.PublishedPort)
		// Always show target port if available
		if port.TargetPort != 0 {
			portStr += fmt.Sprintf("->%d", port.TargetPort)
		}
		// Always append protocol
		protocol := string(port.Protocol)
		if protocol == "" {
			protocol = "tcp"
		}
		portStr += "/" + protocol
		ports = append(ports, portStr)
	}

	if len(ports) == 0 {
		return "-"
	}

	result := strings.Join(ports, ",")
	// Truncate if too long
	if len(result) > 30 {
		return result[:27] + "..."
	}
	return result
}
