package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ContextInfo represents a Docker context with its metadata
type ContextInfo struct {
	Name        string
	Current     bool
	Description string
	DockerHost  string
	TLS         bool
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

				// Check if TLS is enabled by inspecting the context
				usesTLS := false
				if inspectJSON, err := InspectContext(ctx.Name); err == nil {
					// Parse as array first since docker context inspect returns an array
					var inspectArray []map[string]interface{}
					if err := json.Unmarshal([]byte(inspectJSON), &inspectArray); err == nil && len(inspectArray) > 0 {
						inspectData := inspectArray[0]
						// Check for TLSMaterial field (top level)
						if tlsMaterial, ok := inspectData["TLSMaterial"].(map[string]interface{}); ok {
							if dockerTLS, ok := tlsMaterial["docker"]; ok && dockerTLS != nil {
								usesTLS = true
							}
						}
						// Also check legacy TLSData field in Endpoints
						if !usesTLS {
							if endpoints, ok := inspectData["Endpoints"].(map[string]interface{}); ok {
								if docker, ok := endpoints["docker"].(map[string]interface{}); ok {
									if tlsData, ok := docker["TLSData"]; ok && tlsData != nil {
										usesTLS = true
									}
								}
							}
						}
					}
				}

				contexts = append(contexts, ContextInfo{
					Name:        ctx.Name,
					Current:     ctx.Current,
					Description: ctx.Description,
					DockerHost:  ctx.DockerEndpoint,
					TLS:         usesTLS,
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

// CreateContext creates a new Docker context with the given name and Docker host
func CreateContext(name, dockerHost string) error {
	return CreateContextWithTLS(name, dockerHost, "", false)
}

// CreateContextWithTLS creates a new Docker context with optional TLS configuration
func CreateContextWithTLS(name, dockerHost, tlsPath string, skipTLSVerify bool) error {
	if name == "" {
		return fmt.Errorf("context name is required")
	}
	if dockerHost == "" {
		return fmt.Errorf("docker host is required")
	}

	args := []string{"context", "create", name, "--docker", "host=" + dockerHost}

	// Add TLS options if path is provided
	if tlsPath != "" {
		args = append(args, "--docker", "ca="+tlsPath+"/ca.pem")
		args = append(args, "--docker", "cert="+tlsPath+"/cert.pem")
		args = append(args, "--docker", "key="+tlsPath+"/key.pem")
	}

	if skipTLSVerify {
		args = append(args, "--docker", "skip-tls-verify=true")
	}

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Include Docker's error message if available
		if len(output) > 0 {
			return fmt.Errorf("failed to create context %s: %s", name, string(output))
		}
		return fmt.Errorf("failed to create context %s: %w", name, err)
	}

	return nil
}

// CreateContextWithCertFiles creates a Docker context with specific certificate file paths
func CreateContextWithCertFiles(name, description, dockerHost, caFile, certFile, keyFile string, skipTLSVerify bool) error {
	if name == "" {
		return fmt.Errorf("context name is required")
	}
	if dockerHost == "" {
		return fmt.Errorf("docker host is required")
	}

	// Validate certificate files exist if provided
	if caFile != "" || certFile != "" || keyFile != "" {
		if caFile == "" {
			return fmt.Errorf("CA certificate file is required when using TLS")
		}
		if certFile == "" {
			return fmt.Errorf("client certificate file is required when using TLS")
		}
		if keyFile == "" {
			return fmt.Errorf("client key file is required when using TLS")
		}

		// Check if files exist and are readable
		if _, err := os.Stat(caFile); err != nil {
			return fmt.Errorf("CA file not found or not readable: %s", caFile)
		}
		if _, err := os.Stat(certFile); err != nil {
			return fmt.Errorf("certificate file not found or not readable: %s", certFile)
		}
		if _, err := os.Stat(keyFile); err != nil {
			return fmt.Errorf("key file not found or not readable: %s", keyFile)
		}
	}

	args := []string{"context", "create", name}

	// Add description if provided
	if description != "" {
		args = append(args, "--description", description)
	}

	// Build docker endpoint configuration
	dockerConfig := "host=" + dockerHost

	// Add TLS options with individual cert files
	if caFile != "" && certFile != "" && keyFile != "" {
		dockerConfig += ",ca=" + caFile
		dockerConfig += ",cert=" + certFile
		dockerConfig += ",key=" + keyFile
	}

	if skipTLSVerify {
		dockerConfig += ",skip-tls-verify=true"
	}

	args = append(args, "--docker", dockerConfig)

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Include Docker's error message if available
		if len(output) > 0 {
			// Clean up Docker's error message
			errMsg := strings.TrimSpace(string(output))
			errMsg = strings.ReplaceAll(errMsg, "\n", " ")
			return fmt.Errorf("%s", errMsg)
		}
		return fmt.Errorf("failed to create context %s: %w", name, err)
	}

	return nil
}

// UpdateContextDescription updates only the description of a Docker context
func UpdateContextDescription(name, description string) error {
	if name == "" {
		return fmt.Errorf("context name is required")
	}

	args := []string{"context", "update", name}

	// Add description (even if empty, to allow clearing)
	args = append(args, "--description", description)

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Include Docker's error message if available
		if len(output) > 0 {
			// Clean up Docker's error message
			errMsg := strings.TrimSpace(string(output))
			errMsg = strings.ReplaceAll(errMsg, "\n", " ")
			return fmt.Errorf("%s", errMsg)
		}
		return fmt.Errorf("failed to update context %s: %w", name, err)
	}

	return nil
}

// UpdateContextWithCertFiles updates a Docker context with specific certificate file paths
func UpdateContextWithCertFiles(name, description, dockerHost, caFile, certFile, keyFile string, skipTLSVerify bool) error {
	if name == "" {
		return fmt.Errorf("context name is required")
	}

	// Validate certificate files exist if provided
	if caFile != "" || certFile != "" || keyFile != "" {
		if caFile == "" {
			return fmt.Errorf("CA certificate file is required when using TLS")
		}
		if certFile == "" {
			return fmt.Errorf("client certificate file is required when using TLS")
		}
		if keyFile == "" {
			return fmt.Errorf("client key file is required when using TLS")
		}

		// Check if files exist and are readable
		if _, err := os.Stat(caFile); err != nil {
			return fmt.Errorf("CA file not found or not readable: %s", caFile)
		}
		if _, err := os.Stat(certFile); err != nil {
			return fmt.Errorf("certificate file not found or not readable: %s", certFile)
		}
		if _, err := os.Stat(keyFile); err != nil {
			return fmt.Errorf("key file not found or not readable: %s", keyFile)
		}
	}

	args := []string{"context", "update", name}

	// Add description if provided (even if empty, to allow clearing)
	if description != "" {
		args = append(args, "--description", description)
	}

	// Build docker endpoint configuration if host or certs provided
	if dockerHost != "" || caFile != "" {
		dockerConfig := ""

		// Add host if provided
		if dockerHost != "" {
			dockerConfig = "host=" + dockerHost
		}

		// Add TLS options with individual cert files
		if caFile != "" && certFile != "" && keyFile != "" {
			if dockerConfig != "" {
				dockerConfig += ","
			}
			dockerConfig += "ca=" + caFile
			dockerConfig += ",cert=" + certFile
			dockerConfig += ",key=" + keyFile
		}

		if skipTLSVerify {
			if dockerConfig != "" {
				dockerConfig += ","
			}
			dockerConfig += "skip-tls-verify=true"
		}

		if dockerConfig != "" {
			args = append(args, "--docker", dockerConfig)
		}
	}

	cmd := exec.Command("docker", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		// Include Docker's error message if available
		if len(output) > 0 {
			// Clean up Docker's error message
			errMsg := strings.TrimSpace(string(output))
			errMsg = strings.ReplaceAll(errMsg, "\n", " ")
			return fmt.Errorf("%s", errMsg)
		}
		return fmt.Errorf("failed to update context %s: %w", name, err)
	}

	return nil
}
