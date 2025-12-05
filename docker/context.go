package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// ContextInfo represents a Docker context with its metadata
type ContextInfo struct {
	Name        string
	Current     bool
	Description string
	DockerHost  string
	Error       string
}

// ListContexts returns all available Docker contexts using docker CLI
func ListContexts() ([]ContextInfo, error) {
	// Use docker context ls --format json to get context list
	cmd := exec.Command("docker", "context", "ls", "--format", "json")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list contexts: %w", err)
	}

	var contexts []ContextInfo
	// Parse JSON lines (each line is a separate JSON object)
	lines := []byte{}
	for _, b := range output {
		if b == '\n' {
			if len(lines) > 0 {
				var ctx struct {
					Name           string `json:"Name"`
					Current        bool   `json:"Current"`
					Description    string `json:"Description"`
					DockerEndpoint string `json:"DockerEndpoint"`
				}
				if err := json.Unmarshal(lines, &ctx); err != nil {
					return nil, fmt.Errorf("failed to parse context JSON: %w", err)
				}
				contexts = append(contexts, ContextInfo{
					Name:        ctx.Name,
					Current:     ctx.Current,
					Description: ctx.Description,
					DockerHost:  ctx.DockerEndpoint,
				})
				lines = []byte{}
			}
		} else {
			lines = append(lines, b)
		}
	}
	// Handle last line if no trailing newline
	if len(lines) > 0 {
		var ctx struct {
			Name           string `json:"Name"`
			Current        bool   `json:"Current"`
			Description    string `json:"Description"`
			DockerEndpoint string `json:"DockerEndpoint"`
		}
		if err := json.Unmarshal(lines, &ctx); err != nil {
			return nil, fmt.Errorf("failed to parse context JSON: %w", err)
		}
		contexts = append(contexts, ContextInfo{
			Name:        ctx.Name,
			Current:     ctx.Current,
			Description: ctx.Description,
			DockerHost:  ctx.DockerEndpoint,
		})
	}

	return contexts, nil
}

// UseContext switches to the specified Docker context
func UseContext(contextName string) error {
	cmd := exec.Command("docker", "context", "use", contextName)
	// Don't output to stdout/stderr to keep UI clean
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to switch context to %s: %w", contextName, err)
	}
	return nil
}

// ValidateContext checks if a context switch would succeed by attempting to connect
func ValidateContext(contextName string) error {
	// Save current context
	currentCtx, err := GetCurrentContext()
	if err != nil {
		return fmt.Errorf("failed to get current context: %w", err)
	}

	// Try switching to the new context
	if err := UseContext(contextName); err != nil {
		return err
	}

	// Try to create a client and ping
	cli, err := GetClient()
	if err != nil {
		// Switch back to original context
		_ = UseContext(currentCtx)
		return fmt.Errorf("failed to connect to context %s: %w", contextName, err)
	}
	defer func() { _ = cli.Close() }()

	// Verify connection with ping
	ctx := context.Background()
	if _, err := cli.Ping(ctx); err != nil {
		// Switch back to original context
		_ = UseContext(currentCtx)
		return fmt.Errorf("failed to ping context %s: %w", contextName, err)
	}

	return nil
}

// InspectContext returns the detailed JSON inspection of a Docker context
func InspectContext(contextName string) (string, error) {
	cmd := exec.Command("docker", "context", "inspect", "--format", "json", contextName)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to inspect context %s: %w", contextName, err)
	}
	return string(output), nil
}

// ExportContext exports a Docker context to a tar file in /tmp
func ExportContext(contextName string) (string, error) {
	filePath := fmt.Sprintf("/tmp/%s.tar", contextName)
	cmd := exec.Command("docker", "context", "export", contextName, filePath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to export context %s: %w", contextName, err)
	}
	return filePath, nil
}

// ExportContextWithForce exports a Docker context, removing existing file if present
func ExportContextWithForce(contextName string) (string, error) {
	filePath := fmt.Sprintf("/tmp/%s.tar", contextName)
	// Remove existing file if present
	_ = exec.Command("rm", "-f", filePath).Run()

	cmd := exec.Command("docker", "context", "export", contextName, filePath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to export context %s: %w", contextName, err)
	}
	return filePath, nil
}

// CheckContextExportExists checks if an export file already exists for a context
func CheckContextExportExists(contextName string) bool {
	filePath := fmt.Sprintf("/tmp/%s.tar", contextName)
	cmd := exec.Command("test", "-f", filePath)
	return cmd.Run() == nil
}

// DeleteContext removes a Docker context
func DeleteContext(contextName string) error {
	cmd := exec.Command("docker", "context", "rm", contextName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to delete context %s: %w", contextName, err)
	}
	return nil
}

// ImportContext imports a Docker context from a tar file
// Returns the name of the imported context
func ImportContext(filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("file path is required")
	}

	// Extract context name from filename
	parts := strings.Split(filePath, "/")
	fileName := parts[len(parts)-1]
	contextName := fileName
	if idx := len(contextName) - 4; idx > 0 && contextName[idx:] == ".tar" {
		contextName = contextName[:idx]
	}

	cmd := exec.Command("docker", "context", "import", contextName, filePath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to import context from %s: %w", filePath, err)
	}

	return contextName, nil
}
