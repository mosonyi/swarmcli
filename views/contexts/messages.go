// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package contexts

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"swarmcli/docker"
	swarmlog "swarmcli/utils/log"
	inspectview "swarmcli/views/inspect"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

// isContextArchive checks if a filename is a context archive (supports multiple formats)
func isContextArchive(filename string) bool {
	lowerName := strings.ToLower(filename)
	for _, ext := range docker.ContextArchiveExts {
		if strings.HasSuffix(lowerName, ext) {
			return true
		}
	}
	return false
}

type ContextsLoadedMsg struct {
	Contexts []docker.ContextInfo
	Error    error
}

type ContextSwitchedMsg struct {
	ContextName string
	Success     bool
	Error       error
}

type ContextExportedMsg struct {
	ContextName string
	FilePath    string
	Success     bool
	Error       error
}

type ContextImportedMsg struct {
	ContextName string
	Success     bool
	Error       error
}

type ContextDeletedMsg struct {
	ContextName string
	Success     bool
	Error       error
}

type ContextCreatedMsg struct {
	ContextName string
	Success     bool
	Error       error
}

type ContextUpdatedMsg struct {
	ContextName string
	Success     bool
	Error       error
}

// ContextChangedNotification is sent to notify the app that the Docker context has changed
// and should navigate to stacks view
type ContextChangedNotification struct {
	PreviousContext string
}

// loadContextsCmd loads all Docker contexts
func (m *Model) loadContextsCmd() func() tea.Msg {
	contextOps := m.deps.Contexts
	return func() tea.Msg {
		contexts, err := contextOps.ListContexts()
		// Log the result so we can diagnose environments where the CLI is slow
		// or returns an error even though `docker context ls` seems fast.
		l := swarmlog.L()
		if err != nil {
			l.Warnw("ListContexts failed", "error", err)
		} else {
			l.Infow("ListContexts succeeded", "count", len(contexts))
		}
		// Also emit debug information at debug log level so test runs and CI
		// can be inspected via the standard logger instead of writing to /tmp.
		debug := map[string]any{
			"count": len(contexts),
			"error": nil,
		}
		if err != nil {
			debug["error"] = err.Error()
		}
		if b, jerr := json.Marshal(debug); jerr == nil {
			swarmlog.L().Debugf("[LoadContexts] %s", string(b))
		}
		return ContextsLoadedMsg{
			Contexts: contexts,
			Error:    err,
		}
	}
}

// switchContextCmd switches to a different Docker context and validates it's reachable
func (m *Model) switchContextCmd(contextName string) tea.Cmd {
	contextOps := m.deps.Contexts
	return func() tea.Msg {
		// ValidateContext will switch to the context, verify it's reachable,
		// and switch back to the original if validation fails
		err := contextOps.ValidateContext(contextName)
		return ContextSwitchedMsg{
			ContextName: contextName,
			Success:     err == nil,
			Error:       err,
		}
	}
}

// inspectContextCmd inspects a Docker context and navigates to inspect view
func (m *Model) inspectContextCmd(contextName string) tea.Cmd {
	contextOps := m.deps.Contexts
	return func() tea.Msg {
		inspectContent, err := contextOps.InspectContext(contextName)
		if err != nil {
			inspectContent = "Error inspecting context: " + err.Error()
		}
		return view.NavigateToMsg{
			ViewName: inspectview.ViewName,
			Payload: map[string]interface{}{
				"title": "Context: " + contextName,
				"json":  inspectContent,
			},
		}
	}
}

// exportContextCmd exports a Docker context to a file
func (m *Model) exportContextCmd(contextName string) tea.Cmd {
	contextOps := m.deps.Contexts
	return func() tea.Msg {
		// Check if file already exists
		if contextOps.CheckContextExportExists(contextName) {
			// Return a special message indicating file exists
			return ContextExportedMsg{
				ContextName: contextName,
				FilePath:    fmt.Sprintf("/tmp/%s.tar", contextName),
				Success:     false,
				Error:       fmt.Errorf("file_exists"),
			}
		}
		filePath, err := contextOps.ExportContext(contextName)
		return ContextExportedMsg{
			ContextName: contextName,
			FilePath:    filePath,
			Success:     err == nil,
			Error:       err,
		}
	}
}

// exportContextWithForceCmd exports a context, overwriting existing file
func (m *Model) exportContextWithForceCmd(contextName string) tea.Cmd {
	contextOps := m.deps.Contexts
	return func() tea.Msg {
		filePath, err := contextOps.ExportContextWithForce(contextName)
		return ContextExportedMsg{
			ContextName: contextName,
			FilePath:    filePath,
			Success:     err == nil,
			Error:       err,
		}
	}
}

// importContextCmd imports a Docker context from a file
func (m *Model) importContextCmd(filePath string) tea.Cmd {
	contextOps := m.deps.Contexts
	return func() tea.Msg {
		contextName, err := contextOps.ImportContext(filePath)
		return ContextImportedMsg{
			ContextName: contextName,
			Success:     err == nil,
			Error:       err,
		}
	}
}

// deleteContextCmd deletes a Docker context
func (m *Model) deleteContextCmd(contextName string) tea.Cmd {
	contextOps := m.deps.Contexts
	return func() tea.Msg {
		err := contextOps.DeleteContext(contextName)
		return ContextDeletedMsg{
			ContextName: contextName,
			Success:     err == nil,
			Error:       err,
		}
	}
}

// FilesLoadedMsg contains the list of context archive files in a directory
type FilesLoadedMsg struct {
	Path  string
	Files []string
	Error error
}

// LoadFilesCmd loads context archive files and directories from a path for browsing
func LoadFilesCmd(dirPath string) tea.Cmd {
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
			return FilesLoadedMsg{
				Path:  dirPath,
				Files: files,
				Error: err,
			}
		}

		// Separate directories and context archive files
		var dirs []string
		var contextFiles []string

		for _, entry := range entries {
			if entry.IsDir() {
				// Add directory with trailing slash
				dirs = append(dirs, filepath.Join(dirPath, entry.Name())+"/")
			} else if isContextArchive(entry.Name()) {
				contextFiles = append(contextFiles, filepath.Join(dirPath, entry.Name()))
			}
		}

		// Add directories first, then context archive files
		files = append(files, dirs...)
		files = append(files, contextFiles...)

		return FilesLoadedMsg{
			Path:  dirPath,
			Files: files,
			Error: nil,
		}
	}
}

// createContextWithCertFilesCmd creates a new Docker context with individual cert files
func (m *Model) createContextWithCertFilesCmd(name, description, dockerHost, caFile, certFile, keyFile string, useTLS bool) tea.Cmd {
	contextOps := m.deps.Contexts
	return func() tea.Msg {
		var err error
		if useTLS && caFile != "" && certFile != "" && keyFile != "" {
			err = contextOps.CreateContextWithCertFiles(name, description, dockerHost, caFile, certFile, keyFile, false)
		} else {
			err = contextOps.CreateContext(name, dockerHost)
		}
		return ContextCreatedMsg{
			ContextName: name,
			Success:     err == nil,
			Error:       err,
		}
	}
}

// updateContextDescriptionCmd updates only the description of an existing Docker context
func (m *Model) updateContextDescriptionCmd(name, description string) tea.Cmd {
	contextOps := m.deps.Contexts
	return func() tea.Msg {
		err := contextOps.UpdateContextDescription(name, description)
		return ContextUpdatedMsg{
			ContextName: name,
			Success:     err == nil,
			Error:       err,
		}
	}
}

// LoadCertFilesCmd loads all files from a directory for cert file selection
func LoadCertFilesCmd(dirPath string) tea.Cmd {
	return func() tea.Msg {
		files := []string{}

		// Expand ~ to home directory
		if strings.HasPrefix(dirPath, "~") {
			if homeDir, err := os.UserHomeDir(); err == nil {
				dirPath = strings.Replace(dirPath, "~", homeDir, 1)
			}
		}

		// Clean the path
		dirPath = filepath.Clean(dirPath)

		// Add parent directory entry if not at root
		if dirPath != "/" && dirPath != "" {
			files = append(files, "..")
		}

		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return FilesLoadedMsg{
				Path:  dirPath,
				Files: files,
				Error: err,
			}
		}

		// Separate directories and files
		var dirs []string
		var regularFiles []string

		for _, entry := range entries {
			fullPath := filepath.Join(dirPath, entry.Name())
			if entry.IsDir() {
				dirs = append(dirs, fullPath+"/") // Add trailing slash for directories
			} else {
				regularFiles = append(regularFiles, fullPath)
			}
		}

		// Add directories first, then files
		files = append(files, dirs...)
		files = append(files, regularFiles...)

		return FilesLoadedMsg{
			Path:  dirPath,
			Files: files,
			Error: nil,
		}
	}
}
