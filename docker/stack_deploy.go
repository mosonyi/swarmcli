// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// ValidateStackYAML validates that the provided YAML content is a valid Docker Compose file.
func ValidateStackYAML(content string) error {
	// Parse YAML to ensure it's valid
	var data interface{}
	if err := yaml.Unmarshal([]byte(content), &data); err != nil {
		return fmt.Errorf("invalid YAML syntax: %w", err)
	}

	// Check if it looks like a compose file (should have "services" or "version")
	if m, ok := data.(map[string]interface{}); ok {
		// A valid compose file should have at least "services" or "version"
		hasServices := m["services"] != nil
		hasVersion := m["version"] != nil

		if !hasServices && !hasVersion {
			return fmt.Errorf("invalid compose file: missing 'services' and 'version' keys")
		}

		// If it has services, try to parse them
		if services, ok := m["services"].(map[string]interface{}); ok {
			if len(services) == 0 {
				return fmt.Errorf("invalid compose file: 'services' section is empty")
			}
		}
	} else if data == nil {
		return fmt.Errorf("empty YAML content")
	} else {
		return fmt.Errorf("invalid compose file format")
	}

	return nil
}

// DeployStack deploys a stack with the provided name and YAML content.
func DeployStack(stackName string, yamlContent string) error {
	// Validate first
	if err := ValidateStackYAML(yamlContent); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	if stackName == "" {
		return fmt.Errorf("stack name cannot be empty")
	}

	// Create temporary file with the YAML content
	tmpFile, err := os.CreateTemp("", "swarmcli-stack-*.yml")
	if err != nil {
		return fmt.Errorf("failed to create temporary file: %w", err)
	}
	defer func() {
		_ = os.Remove(tmpFile.Name())
	}()

	// Write YAML content to temp file
	if _, err := tmpFile.WriteString(yamlContent); err != nil {
		_ = tmpFile.Close()
		return fmt.Errorf("failed to write YAML to temporary file: %w", err)
	}
	if err := tmpFile.Close(); err != nil {
		return fmt.Errorf("failed to close temporary file: %w", err)
	}

	// Get Docker context
	ctx, err := GetDockerContext()
	if err != nil {
		return fmt.Errorf("failed to get docker context: %w", err)
	}

	// Execute docker stack deploy command
	cmd := exec.Command("docker", "--context", ctx, "stack", "deploy", "-c", tmpFile.Name(), stackName)
	cmd.Env = os.Environ()

	// Capture output for error reporting
	output, err := cmd.CombinedOutput()
	if err != nil {
		// If deployment failed, try to clean up any networks that might have been created
		// This handles the case where a network with the wrong type was created
		if networkNames := extractNetworkNames(yamlContent); len(networkNames) > 0 {
			l().Infof("Deployment failed, attempting to clean up any orphaned networks")
			cleanupNetworks(stackName, networkNames)
		}
		return fmt.Errorf("failed to deploy stack: %w\nOutput: %s", err, string(output))
	}

	l().Infof("Stack %q deployed successfully", stackName)
	return nil
}

// RemoveStackCLI tears down a stack via `docker stack rm`, the symmetric
// counterpart to DeployStack. Unlike RemoveStack (services only), this removes
// the stack's services, networks, configs and secrets while leaving volumes
// intact — matching standard Docker stack semantics.
func RemoveStackCLI(stackName string) error {
	if stackName == "" {
		return fmt.Errorf("stack name cannot be empty")
	}
	ctx, err := GetDockerContext()
	if err != nil {
		return fmt.Errorf("failed to get docker context: %w", err)
	}
	cmd := exec.Command("docker", "--context", ctx, "stack", "rm", stackName)
	cmd.Env = os.Environ()
	output, err := cmd.CombinedOutput()
	if err != nil {
		// An already-absent stack is success: makes uninstall idempotent so a
		// retry after a partial teardown can still finish cleanup.
		if strings.Contains(string(output), "Nothing found in stack") {
			l().Infof("Stack %q already absent", stackName)
			return nil
		}
		return fmt.Errorf("failed to remove stack: %w\nOutput: %s", err, string(output))
	}
	l().Infof("Stack %q removed", stackName)
	return nil
}

// extractNetworkNames extracts network names from a compose YAML file
func extractNetworkNames(yamlContent string) []string {
	var data map[string]interface{}
	if err := yaml.Unmarshal([]byte(yamlContent), &data); err != nil {
		return nil
	}

	var networkNames []string
	if networks, ok := data["networks"].(map[string]interface{}); ok {
		for name := range networks {
			networkNames = append(networkNames, name)
		}
	}
	return networkNames
}

// cleanupNetworks attempts to remove networks that may have been created during a failed deployment
func cleanupNetworks(stackName string, networkNames []string) {
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	for _, netName := range networkNames {
		// Docker creates networks with the pattern: stackname_networkname
		fullNetworkName := fmt.Sprintf("%s_%s", stackName, netName)
		l().Infof("Attempting to clean up network: %s", fullNetworkName)

		// Try to remove the network (non-forced)
		if err := RemoveNetwork(ctx, fullNetworkName); err != nil {
			l().Debugf("Could not remove network %s: %v (this may be expected if it wasn't created)", fullNetworkName, err)
		} else {
			l().Infof("Successfully cleaned up orphaned network: %s", fullNetworkName)
		}
	}
}

// GetDockerContext returns the current Docker context name.
func GetDockerContext() (string, error) {
	// Check DOCKER_CONTEXT environment variable first
	if ctx := os.Getenv("DOCKER_CONTEXT"); ctx != "" {
		return ctx, nil
	}

	// Fall back to running "docker context show"
	output, err := exec.Command("docker", "context", "show").Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current docker context: %w", err)
	}

	// Parse output and return trimmed context name
	ctx := string(output)
	if len(ctx) > 0 && ctx[len(ctx)-1] == '\n' {
		ctx = ctx[:len(ctx)-1]
	}
	return ctx, nil
}
