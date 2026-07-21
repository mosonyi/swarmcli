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

// ResolveImage selects how the daemon resolves image tags to digests at deploy
// time, mirroring `docker stack deploy --resolve-image`.
//
// The empty value passes no flag, leaving Docker's own default of "always": the
// manager queries the registry for every service on every deploy. That makes a
// deploy fail when the registry is unreachable even though every node already
// has the image, and it rewrites the spec to repo:tag@sha256:..., so anything
// diffing desired against live sees every service as changed.
//
// "changed" re-resolves only when the compose file's image string differs from
// the com.docker.stack.image label the CLI stashes, which is what a reconciler
// wants. "never" has an open upstream rollout bug (moby#51658).
type ResolveImage string

const (
	ResolveImageDefault ResolveImage = ""
	ResolveImageAlways  ResolveImage = "always"
	ResolveImageChanged ResolveImage = "changed"
	ResolveImageNever   ResolveImage = "never"
)

// Valid reports whether r is a mode the docker CLI accepts.
func (r ResolveImage) Valid() bool {
	switch r {
	case ResolveImageDefault, ResolveImageAlways, ResolveImageChanged, ResolveImageNever:
		return true
	}
	return false
}

// DeployStack deploys a stack with the provided name and YAML content, leaving
// image resolution at Docker's default. See DeployStackResolved.
func DeployStack(stackName string, yamlContent string) error {
	return DeployStackResolved(stackName, yamlContent, ResolveImageDefault)
}

// DeployStackResolved deploys a stack with an explicit image-resolution mode,
// on whichever context the process is pointed at.
func DeployStackResolved(stackName string, yamlContent string, resolve ResolveImage) error {
	ctxName, err := GetDockerContext()
	if err != nil {
		return fmt.Errorf("failed to get docker context: %w", err)
	}
	return DeployStackInContext(ctxName, stackName, yamlContent, resolve)
}

// DeployStackInContext deploys a stack to an explicitly named Docker context.
//
// `docker stack deploy` has no SDK equivalent, so this shells out; naming the
// context is what keeps the target an argument rather than whatever
// DOCKER_CONTEXT or `docker context show` happens to say at the moment the
// command runs.
func DeployStackInContext(ctxName, stackName, yamlContent string, resolve ResolveImage) error {
	if ctxName == "" {
		return fmt.Errorf("docker context name is required")
	}
	if !resolve.Valid() {
		return fmt.Errorf("invalid --resolve-image %q: want always, changed or never", string(resolve))
	}
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

	// Execute docker stack deploy command
	args := []string{"--context", ctxName, "stack", "deploy", "-c", tmpFile.Name()}
	if resolve != ResolveImageDefault {
		args = append(args, "--resolve-image", string(resolve))
	}
	args = append(args, stackName)
	cmd := exec.Command("docker", args...)
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
	ctxName, err := GetDockerContext()
	if err != nil {
		return fmt.Errorf("failed to get docker context: %w", err)
	}
	return RemoveStackCLIInContext(ctxName, stackName)
}

// RemoveStackCLIInContext is RemoveStackCLI against an explicitly named context.
func RemoveStackCLIInContext(ctxName, stackName string) error {
	if ctxName == "" {
		return fmt.Errorf("docker context name is required")
	}
	if stackName == "" {
		return fmt.Errorf("stack name cannot be empty")
	}
	cmd := exec.Command("docker", "--context", ctxName, "stack", "rm", stackName)
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
