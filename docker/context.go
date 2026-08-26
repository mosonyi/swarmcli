// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package docker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ContextArchiveExts lists supported Docker context archive extensions.
// Keep multi-part extensions before shorter suffixes when order matters.
var ContextArchiveExts = []string{".dockercontext", ".tar.gz", ".tgz", ".tar"}

// ContextInfo represents a Docker context with its metadata
type ContextInfo struct {
	Name        string
	Current     bool
	Description string
	DockerHost  string
	TLS         bool
	Error       string
}

// DefaultContextName is Docker's built-in context.
const DefaultContextName = "default"

// ErrDefaultContextImmutable reports an update aimed at the built-in context.
// `default` is a name the context store reserves and refuses to write
// (docker/cli cli/context/store.ValidateContextName), so such an update can
// only fail.
var ErrDefaultContextImmutable = errors.New(
	"the default context cannot be edited — create a context to point at another host")

// dockerEndpointName is the key Docker files a context's Docker endpoint
// under, both in `docker context inspect` output and as the directory holding
// that endpoint's TLS material inside the context store.
const dockerEndpointName = "docker"

// The names the context store writes its copies of the TLS material under
// (docker/cli cli/context/tlsdata.go).
const (
	tlsCAFile   = "ca.pem"
	tlsCertFile = "cert.pem"
	tlsKeyFile  = "key.pem"
)

// ContextEndpoint is a context's stored Docker endpoint. The cert fields point
// at the context store's own copies of the TLS material: the store keeps the
// bytes rather than the paths a context was created from, so those copies are
// the only reference an update can re-supply. They are empty for a context
// with no TLS material, such as an ssh:// endpoint.
type ContextEndpoint struct {
	Host          string
	SkipTLSVerify bool
	CAFile        string
	CertFile      string
	KeyFile       string
}

// HasTLS reports whether the endpoint carries a complete set of TLS material.
func (e ContextEndpoint) HasTLS() bool {
	return e.CAFile != "" && e.CertFile != "" && e.KeyFile != ""
}

// contextEndpointInspect is the subset of `docker context inspect` describing
// the Docker endpoint and where the context store keeps its TLS material.
type contextEndpointInspect struct {
	Endpoints map[string]struct {
		Host          string `json:"Host"`
		SkipTLSVerify bool   `json:"SkipTLSVerify"`
	} `json:"Endpoints"`
	TLSMaterial map[string][]string `json:"TLSMaterial"`
	Storage     struct {
		TLSPath string `json:"TLSPath"`
	} `json:"Storage"`
}

// contextListItem represents a single context from docker context ls --format json
type contextListItem struct {
	Name           string `json:"Name"`
	Current        bool   `json:"Current"`
	Description    string `json:"Description"`
	DockerEndpoint string `json:"DockerEndpoint"`
}

// contextInspectResult represents the structure from docker context inspect
type contextInspectResult struct {
	TLSMaterial struct {
		Docker interface{} `json:"docker"`
	} `json:"TLSMaterial"`
	Endpoints struct {
		Docker struct {
			TLSData interface{} `json:"TLSData"`
		} `json:"docker"`
	} `json:"Endpoints"`
}

// ListContexts returns all available Docker contexts using docker CLI.
//
// `docker context ls` marks whichever context ~/.docker/config.json says is
// current, which is not necessarily the one this session is talking to. The
// current flag is recomputed against the session pin: a list that marked the
// externally-switched context current while every read came from the pinned
// one would be reporting the very mismatch #611 is about. The pin is also
// where the contexts view reads the context being left when a switch is
// confirmed.
func ListContexts() ([]ContextInfo, error) {
	output, err := listContextsFn()
	if err != nil {
		return nil, fmt.Errorf("failed to list contexts: %w", err)
	}

	// An unresolvable pin leaves every entry unmarked rather than failing the
	// list: the names are still worth showing, and a switch is how the user
	// would recover.
	sessionName, _ := SessionContext()

	var contexts []ContextInfo
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		var item contextListItem
		if err := json.Unmarshal(scanner.Bytes(), &item); err != nil {
			return nil, fmt.Errorf("failed to parse context JSON: %w", err)
		}

		contexts = append(contexts, ContextInfo{
			Name:        item.Name,
			Current:     item.Name == sessionName,
			Description: item.Description,
			DockerHost:  item.DockerEndpoint,
			TLS:         checkContextTLS(item.Name),
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to read context list: %w", err)
	}

	return contexts, nil
}

// listContextsFn is the seam for `docker context ls`, so which entry ListContexts
// marks current is testable without a Docker CLI on PATH.
var listContextsFn = runContextList

func runContextList() ([]byte, error) {
	return exec.Command("docker", "context", "ls", "--format", "json").Output()
}

// checkContextTLS checks if a context has TLS enabled
func checkContextTLS(contextName string) bool {
	inspectJSON, err := InspectContext(contextName)
	if err != nil {
		return false
	}

	var inspectArray []contextInspectResult
	if err := json.Unmarshal([]byte(inspectJSON), &inspectArray); err != nil || len(inspectArray) == 0 {
		return false
	}

	inspect := inspectArray[0]

	// Check for TLSMaterial field (current format)
	if inspect.TLSMaterial.Docker != nil {
		return true
	}

	// Check legacy TLSData field in Endpoints
	if inspect.Endpoints.Docker.TLSData != nil {
		return true
	}

	return false
}

// validateTLSFiles validates that all three TLS certificate files are provided and exist
func validateTLSFiles(caFile, certFile, keyFile string) error {
	// If any TLS file is provided, all three must be provided
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
	return nil
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

// ErrContextPinnedByEnv reports a switch that could not take effect because
// DOCKER_CONTEXT names a different context. `docker context use` writes the
// active context to ~/.docker/config.json, but the variable is resolved ahead
// of it, so the switch would change nothing this process observes.
var ErrContextPinnedByEnv = errors.New("DOCKER_CONTEXT pins the Docker context")

// Seams for ValidateContext, so its switch/revert bookkeeping is testable
// without a Docker daemon. probeContextFn is the half that needs one.
var (
	useContextFn     = UseContext
	resetClientFn    = ResetClient
	currentContextFn = GetCurrentContext
	setContextFn     = SetSessionContext
	probeContextFn   = probeActiveContext
)

// ValidateContext checks if a context switch would succeed by attempting to connect
func ValidateContext(ctx context.Context, contextName string) error {
	// A switch cannot take effect while DOCKER_CONTEXT names something else,
	// so report that instead of writing config.json and claiming success.
	// Naming the same context is not a conflict — nothing has to move.
	if env := envContext(); env != "" && env != contextName {
		return fmt.Errorf("%w to '%s' — unset it to switch to '%s'", ErrContextPinnedByEnv, env, contextName)
	}

	// Save current context
	currentCtx, err := currentContextFn()
	if err != nil {
		return fmt.Errorf("failed to get current context: %w", err)
	}

	// Try switching to the new context
	if err := useContextFn(contextName); err != nil {
		return err
	}
	// Two pieces of state, both stale until moved: the session pin every
	// caller resolves its context through, and the process-wide client
	// singleton built for the context we just left. Without dropping the
	// client, every check below would describe that context and pass whatever
	// the new one is doing; without moving the pin, the probe below would
	// build the replacement for the old context all over again.
	setContextFn(contextName)
	resetClientFn()

	if err := probeContextFn(ctx, contextName); err != nil {
		// Switch back to original context, and drop the client built for the
		// one being rejected.
		_ = useContextFn(currentCtx)
		setContextFn(currentCtx)
		resetClientFn()
		return err
	}
	return nil
}

// probeActiveContext connects to whichever context is now active and reports
// whether it is reachable and part of a usable swarm. contextName names it in
// the error text only.
func probeActiveContext(ctx context.Context, contextName string) error {
	cli, err := GetClient()
	if err != nil {
		return fmt.Errorf("failed to connect to context %s: %w", contextName, err)
	}
	// Verify connection with ping
	if _, err := cli.Ping(ctx); err != nil {
		return fmt.Errorf("failed to ping context %s: %w", contextName, err)
	}

	// Verify the node is part of a Swarm cluster
	info, err := cli.Info(ctx)
	if err != nil {
		// A locked swarm is reachable, just encrypted — allow the switch so the
		// user can unlock it from within swarmcli (CLI parity with `context use`).
		if IsSwarmLockedErr(err) {
			return nil
		}
		return fmt.Errorf("failed to query info for context %s: %w", contextName, err)
	}
	if !isUsableSwarmState(info.Swarm.LocalNodeState) {
		return fmt.Errorf("context %s is not part of a Docker Swarm cluster", contextName)
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

// InspectContextEndpoint returns the Docker endpoint stored for a context.
func InspectContextEndpoint(contextName string) (ContextEndpoint, error) {
	inspectJSON, err := InspectContext(contextName)
	if err != nil {
		return ContextEndpoint{}, err
	}
	return parseContextEndpoint(contextName, inspectJSON)
}

// parseContextEndpoint reads a single context out of `docker context inspect`
// output, resolving its TLS material to paths inside the context store.
func parseContextEndpoint(contextName, inspectJSON string) (ContextEndpoint, error) {
	var inspected []contextEndpointInspect
	if err := json.Unmarshal([]byte(inspectJSON), &inspected); err != nil {
		return ContextEndpoint{}, fmt.Errorf("failed to parse inspect output for context '%s': %w", contextName, err)
	}
	if len(inspected) == 0 {
		return ContextEndpoint{}, fmt.Errorf("context '%s' was not found", contextName)
	}

	meta := inspected[0].Endpoints[dockerEndpointName]
	endpoint := ContextEndpoint{Host: meta.Host, SkipTLSVerify: meta.SkipTLSVerify}

	tlsDir := filepath.Join(inspected[0].Storage.TLSPath, dockerEndpointName)
	for _, file := range inspected[0].TLSMaterial[dockerEndpointName] {
		switch file {
		case tlsCAFile:
			endpoint.CAFile = filepath.Join(tlsDir, file)
		case tlsCertFile:
			endpoint.CertFile = filepath.Join(tlsDir, file)
		case tlsKeyFile:
			endpoint.KeyFile = filepath.Join(tlsDir, file)
		}
	}
	return endpoint, nil
}

// ExportContext exports a Docker context to a tar file in /tmp
func ExportContext(contextName string) (string, error) {
	filePath := fmt.Sprintf("/tmp/%s.tar", filepath.Base(contextName))
	cmd := exec.Command("docker", "context", "export", contextName, filePath)
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("failed to export context %s: %w", contextName, err)
	}
	return filePath, nil
}

// ExportContextWithForce exports a Docker context, removing existing file if present
func ExportContextWithForce(contextName string) (string, error) {
	filePath := fmt.Sprintf("/tmp/%s.tar", filepath.Base(contextName))
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
	filePath := fmt.Sprintf("/tmp/%s.tar", filepath.Base(contextName))
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

// ImportContext imports a Docker context from an archive file
// Returns the name of the imported context
func ImportContext(filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("file path is required")
	}

	// Extract context name from filename and support multiple archive types
	fileName := filepath.Base(filePath)
	contextName := fileName

	// Remove known extensions if present (order matters for multi-part extensions)
	for _, ext := range ContextArchiveExts {
		if strings.HasSuffix(strings.ToLower(contextName), ext) {
			contextName = contextName[:len(contextName)-len(ext)]
			break
		}
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

	// Validate certificate files
	if err := validateTLSFiles(caFile, certFile, keyFile); err != nil {
		return err
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

// UpdateContextEndpoint updates a Docker context's description and host. An
// empty value leaves that field as it is.
//
// Changing the host carries the context's existing TLS material forward.
// `docker context update` does not merge: it replaces the whole endpoint and
// resets the stored TLS material to exactly what --docker names, so a bare
// host= would delete a TLS context's certificates (docker/cli
// cli/command/context/update.go).
//
// Docker ignores an empty --description, so a description cannot be cleared
// this way. A caller that lets a user empty the field must say so rather than
// report a success that changed nothing.
func UpdateContextEndpoint(name, description, dockerHost string) error {
	if name == "" {
		return fmt.Errorf("context name is required")
	}
	if name == DefaultContextName {
		return ErrDefaultContextImmutable
	}

	// Only a host change replaces the endpoint, so only then is the stored
	// material at stake — and worth an inspect.
	var endpoint ContextEndpoint
	if dockerHost != "" {
		var err error
		endpoint, err = InspectContextEndpoint(name)
		if err != nil {
			return err
		}
	}

	return runContextUpdate(name, updateContextArgs(name, description, dockerHost, endpoint))
}

// updateContextArgs builds the argv for `docker context update`. endpoint
// supplies the TLS material to re-state alongside a new host; it is ignored
// when the host is unchanged, because then no --docker is passed and Docker
// leaves the endpoint alone.
func updateContextArgs(name, description, dockerHost string, endpoint ContextEndpoint) []string {
	args := []string{"context", "update", name}

	if description != "" {
		args = append(args, "--description", description)
	}
	if dockerHost == "" {
		return args
	}

	dockerConfig := "host=" + dockerHost
	if endpoint.HasTLS() {
		dockerConfig += ",ca=" + endpoint.CAFile
		dockerConfig += ",cert=" + endpoint.CertFile
		dockerConfig += ",key=" + endpoint.KeyFile
	}
	if endpoint.SkipTLSVerify {
		dockerConfig += ",skip-tls-verify=true"
	}
	return append(args, "--docker", dockerConfig)
}

// runContextUpdate runs `docker context update`, preferring Docker's own error
// text over the exit status.
func runContextUpdate(name string, args []string) error {
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
