package secretsview

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"swarmcli/core/primitives/hash"
	"swarmcli/docker"
	inspectview "swarmcli/views/inspect"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/swarm"
)

// --- Async commands ---

func loadSecretsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		secs, err := docker.ListSecrets(ctx)
		if err != nil {
			return errorMsg(fmt.Errorf("failed to list secrets: %w", err))
		}

		wrapped := make([]docker.SecretWithDecodedData, len(secs))
		for i, s := range secs {
			wrapped[i] = docker.SecretWithDecodedData{Secret: s, Data: nil}
		}
		return secretsLoadedMsg(wrapped)
	}
}

// computeSecretUsedCmd checks which secrets are used by services in background
// and returns a usedStatusUpdatedMsg containing a map[id]bool.
func computeSecretUsedCmd(secs []docker.SecretWithDecodedData) tea.Cmd {
	return func() tea.Msg {
		usedMap := make(map[string]bool, len(secs))
		ctx := context.Background()
		for _, s := range secs {
			usedMap[s.Secret.ID] = false
			svcs, err := docker.ListServicesUsingSecretID(ctx, s.Secret.ID)
			if err == nil && len(svcs) > 0 {
				usedMap[s.Secret.ID] = true
			}
		}
		return usedStatusUpdatedMsg(usedMap)
	}
}

// CheckSecretsCmd checks if secrets have changed and returns update message if so
func CheckSecretsCmd(lastHash uint64) tea.Cmd {
	return func() tea.Msg {
		l().Info("CheckSecretsCmd: Polling for secret changes")

		ctx := context.Background()
		secs, err := docker.ListSecrets(ctx)
		if err != nil {
			l().Errorf("CheckSecretsCmd: ListSecrets failed: %v", err)
			return tickCmd()
		}

		wrapped := make([]docker.SecretWithDecodedData, len(secs))
		for i, s := range secs {
			wrapped[i] = docker.SecretWithDecodedData{Secret: s, Data: nil}
		}

		// Create a stable hash based only on ID and Version (not timestamps)
		type stableSecret struct {
			ID      string
			Version uint64
			Name    string
		}
		stableSecrets := make([]stableSecret, len(secs))
		for i, s := range secs {
			stableSecrets[i] = stableSecret{
				ID:      s.ID,
				Version: s.Version.Index,
				Name:    s.Spec.Name,
			}
		}

		newHash, err := hash.Compute(stableSecrets)
		if err != nil {
			l().Errorf("CheckSecretsCmd: Error computing hash: %v", err)
			// Schedule next poll even on error
			return tickCmd()
		}

		l().Infof("CheckSecretsCmd: lastHash=%s, newHash=%s, secretCount=%d",
			hash.Fmt(lastHash), hash.Fmt(newHash), len(wrapped))

		// Only return update message if something changed
		if newHash != lastHash {
			l().Info("CheckSecretsCmd: Change detected! Refreshing secret list")
			return secretsLoadedMsg(wrapped)
		}

		l().Info("CheckSecretsCmd: No changes detected, scheduling next poll")
		// Schedule next poll in 5 seconds
		return tickCmd()
	}
}

func inspectSecretCmd(name string) tea.Cmd {
	return func() tea.Msg {
		sec, err := docker.InspectSecret(context.Background(), name)
		jsonStr := ""
		if err != nil {
			jsonStr = fmt.Sprintf("Error inspecting secret %q: %v", name, err)
		} else if data, err := sec.JSON(); err != nil {
			jsonStr = fmt.Sprintf("Error marshalling secret %q: %v", name, err)
		} else {
			jsonStr = string(data)
		}

		return view.NavigateToMsg{
			ViewName: inspectview.ViewName,
			Payload: map[string]interface{}{
				"title": fmt.Sprintf("Secret: %s", name),
				"json":  jsonStr,
				"meta": map[string]interface{}{
					"ID":   sec.Secret.ID,
					"Name": sec.Secret.Spec.Name,
					"Note": "Secret data is not available (write-only)",
				},
			},
		}
	}
}

func pushRevealViewCmd(name string) tea.Cmd {
	return func() tea.Msg {
		// Push a reveal view that will fetch the secret content
		return view.NavigateToMsg{
			ViewName: "reveal-secret",
			Payload: map[string]interface{}{
				"secretName": name,
			},
		}
	}
}

func deleteSecretCmd(name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := docker.DeleteSecret(ctx, name)
		if err != nil {
			return errorMsg(fmt.Errorf("failed to delete secret %q: %w", name, err))
		}
		return secretDeletedMsg{Name: name}
	}
}

func loadFilesCmd(dirPath string) tea.Cmd {
	return func() tea.Msg {
		files := []string{}

		// Expand ~ to home directory
		if strings.HasPrefix(dirPath, "~") {
			if homeDir, err := os.UserHomeDir(); err == nil {
				dirPath = strings.Replace(dirPath, "~", homeDir, 1)
			}
		}

		// Add parent directory option if not root
		if dirPath != "/" {
			files = append(files, "..")
		}

		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return fileBrowserMsg{
				Path:  dirPath,
				Files: files,
			}
		}

		// Separate directories and regular files
		var dirs []string
		var regFiles []string

		for _, entry := range entries {
			if entry.IsDir() {
				// Add directory with trailing slash
				dirs = append(dirs, filepath.Join(dirPath, entry.Name())+"/")
			} else {
				// Add all files
				regFiles = append(regFiles, filepath.Join(dirPath, entry.Name()))
			}
		}

		// Add directories first, then files
		files = append(files, dirs...)
		files = append(files, regFiles...)

		return fileBrowserMsg{
			Path:  dirPath,
			Files: files,
		}
	}
}

func createSecretFromFileCmd(name, filePath string, labels map[string]string, encode bool) tea.Cmd {
	return func() tea.Msg {
		l().Infof("Creating secret %s from file %s (encode=%v, labels=%v)", name, filePath, encode, labels)

		// Read file content
		data, err := os.ReadFile(filePath)
		if err != nil {
			l().Errorf("Failed to read file %s: %v", filePath, err)
			return errorMsg(fmt.Errorf("failed to read file: %w", err))
		}

		// Base64 encode if requested
		if encode {
			encoded := base64.StdEncoding.EncodeToString(data)
			// Debug: show encoded payload (may be large)
			const maxLogged = 4096
			if len(encoded) > maxLogged {
				l().Debugf("Secret %s base64 payload (%d chars, truncated to %d): %s…", name, len(encoded), maxLogged, encoded[:maxLogged])
			} else {
				l().Debugf("Secret %s base64 payload (%d chars): %s", name, len(encoded), encoded)
			}
			data = []byte(encoded)
		}

		// Create the secret
		ctx := context.Background()
		newSec, err := docker.CreateSecret(ctx, name, data, labels)
		if err != nil {
			l().Errorf("Failed to create secret %s: %v", name, err)
			return errorMsg(fmt.Errorf("failed to create secret: %w", err))
		}

		l().Infof("Successfully created secret %s from file", name)
		return secretCreatedMsg{
			Name:   name,
			Secret: docker.SecretWithDecodedData{Secret: newSec, Data: nil},
		}
	}
}

func createSecretFromContentCmd(name string, content []byte, labels map[string]string, encode bool) tea.Cmd {
	return func() tea.Msg {
		l().Infof("Creating secret %s from inline content (encode=%v, labels=%v)", name, encode, labels)

		// Base64 encode if requested
		if encode {
			encoded := base64.StdEncoding.EncodeToString(content)
			// Debug: show encoded payload
			const maxLogged = 4096
			if len(encoded) > maxLogged {
				l().Debugf("Secret %s base64 payload (%d chars, truncated to %d): %s…", name, len(encoded), maxLogged, encoded[:maxLogged])
			} else {
				l().Debugf("Secret %s base64 payload (%d chars): %s", name, len(encoded), encoded)
			}
			content = []byte(encoded)
		}

		// Create the secret
		ctx := context.Background()
		newSec, err := docker.CreateSecret(ctx, name, content, labels)
		if err != nil {
			l().Errorf("Failed to create secret %s: %v", name, err)
			return errorMsg(fmt.Errorf("failed to create secret: %w", err))
		}

		l().Infof("Successfully created secret %s", name)
		return secretCreatedMsg{
			Name:   name,
			Secret: docker.SecretWithDecodedData{Secret: newSec, Data: nil},
		}
	}
}

func getUsedByStacksCmd(secretName string) tea.Cmd {
	return func() tea.Msg {
		l().Infof("Getting stacks/services that use secret: %s", secretName)

		ctx := context.Background()
		// Get secret ID for robust matching
		sec, err := docker.InspectSecret(ctx, secretName)
		if err != nil {
			l().Errorf("Failed to inspect secret %s: %v", secretName, err)
			return errorMsg(err)
		}

		// Get services by secret name and ID
		servicesByName, err := docker.ListServicesUsingSecretName(ctx, secretName)
		if err != nil {
			l().Errorf("Failed to list services using secret name %s: %v", secretName, err)
			return errorMsg(err)
		}
		servicesByID, err := docker.ListServicesUsingSecretID(ctx, sec.Secret.ID)
		if err != nil {
			l().Errorf("Failed to list services using secret ID %s: %v", sec.Secret.ID, err)
			return errorMsg(err)
		}

		// Merge services, avoid duplicates
		svcMap := make(map[string]swarm.Service)
		for _, svc := range servicesByName {
			svcMap[svc.ID] = svc
		}
		for _, svc := range servicesByID {
			svcMap[svc.ID] = svc
		}

		// Collect stack/service pairs
		var usedBy []usedByItem
		for _, svc := range svcMap {
			stackName := svc.Spec.Labels["com.docker.stack.namespace"]
			if stackName == "" {
				stackName = "(no stack)"
			}
			usedBy = append(usedBy, usedByItem{
				StackName:   stackName,
				ServiceName: svc.Spec.Name,
			})
		}

		// Sort by stack then service
		sort.Slice(usedBy, func(i, j int) bool {
			if usedBy[i].StackName == usedBy[j].StackName {
				return usedBy[i].ServiceName < usedBy[j].ServiceName
			}
			return usedBy[i].StackName < usedBy[j].StackName
		})

		l().Infof("Secret %s is used by %d service(s)", secretName, len(usedBy))

		return usedByMsg{SecretName: secretName, UsedBy: usedBy}
	}
}

// openEditorForContentCmd opens the user's editor to edit secret content
func openEditorForContentCmd(initialData string) tea.Cmd {
	l().Infoln("openEditorForContentCmd: started")

	tmp, err := os.CreateTemp("", "secret-*.txt")
	if err != nil {
		l().Infoln("CreateTemp error:", err)
		return func() tea.Msg { return errorMsg(fmt.Errorf("failed to create temp file: %w", err)) }
	}
	defer func(tmp *os.File) {
		_ = tmp.Close()
	}(tmp)

	if initialData != "" {
		if _, err := tmp.WriteString(initialData); err != nil {
			l().Infoln("Write temp error:", err)
			return func() tea.Msg { return errorMsg(fmt.Errorf("failed to write temp file: %w", err)) }
		}
	}

	l().Infoln("Created temp file:", tmp.Name())

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano"
	}

	l().Infoln("Invoking editor:", editor, tmp.Name())
	cmd := exec.Command(editor, tmp.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		// Clean up temp file
		defer func(name string) {
			_ = os.Remove(name)
		}(tmp.Name())

		if err != nil {
			l().Infoln("Editor process error:", err)
			return errorMsg(fmt.Errorf("editor failed: %w", err))
		}

		l().Infoln("Editor closed, reading back from temp")
		newData, err := os.ReadFile(tmp.Name())
		if err != nil {
			l().Infoln("ReadFile error:", err)
			return errorMsg(fmt.Errorf("failed to read edited file: %w", err))
		}

		l().Infoln("Read new data, length:", len(newData))
		return editorContentMsg{Content: string(newData)}
	})
}
