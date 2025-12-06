package contexts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"swarmcli/docker"
	inspectview "swarmcli/views/inspect"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

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
type ContextChangedNotification struct{}

// LoadContextsCmd loads all Docker contexts
func LoadContextsCmd() tea.Msg {
	contexts, err := docker.ListContexts()
	return ContextsLoadedMsg{
		Contexts: contexts,
		Error:    err,
	}
}

// SwitchContextCmd switches to a different Docker context and validates it's reachable
func SwitchContextCmd(contextName string) tea.Cmd {
	return func() tea.Msg {
		// ValidateContext will switch to the context, verify it's reachable,
		// and switch back to the original if validation fails
		err := docker.ValidateContext(contextName)
		return ContextSwitchedMsg{
			ContextName: contextName,
			Success:     err == nil,
			Error:       err,
		}
	}
}

// InspectContextCmd inspects a Docker context and navigates to inspect view
func InspectContextCmd(contextName string) tea.Cmd {
	return func() tea.Msg {
		inspectContent, err := docker.InspectContext(contextName)
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

// ExportContextCmd exports a Docker context to a file
func ExportContextCmd(contextName string) tea.Cmd {
	return func() tea.Msg {
		// Check if file already exists
		if docker.CheckContextExportExists(contextName) {
			// Return a special message indicating file exists
			return ContextExportedMsg{
				ContextName: contextName,
				FilePath:    fmt.Sprintf("/tmp/%s.tar", contextName),
				Success:     false,
				Error:       fmt.Errorf("file_exists"),
			}
		}
		filePath, err := docker.ExportContext(contextName)
		return ContextExportedMsg{
			ContextName: contextName,
			FilePath:    filePath,
			Success:     err == nil,
			Error:       err,
		}
	}
}

// ExportContextWithForceCmd exports a context, overwriting existing file
func ExportContextWithForceCmd(contextName string) tea.Cmd {
	return func() tea.Msg {
		filePath, err := docker.ExportContextWithForce(contextName)
		return ContextExportedMsg{
			ContextName: contextName,
			FilePath:    filePath,
			Success:     err == nil,
			Error:       err,
		}
	}
}

// ImportContextCmd imports a Docker context from a file
func ImportContextCmd(filePath string) tea.Cmd {
	return func() tea.Msg {
		contextName, err := docker.ImportContext(filePath)
		return ContextImportedMsg{
			ContextName: contextName,
			Success:     err == nil,
			Error:       err,
		}
	}
}

// DeleteContextCmd deletes a Docker context
func DeleteContextCmd(contextName string) tea.Cmd {
	return func() tea.Msg {
		err := docker.DeleteContext(contextName)
		return ContextDeletedMsg{
			ContextName: contextName,
			Success:     err == nil,
			Error:       err,
		}
	}
}

// FilesLoadedMsg contains the list of tar files in a directory
type FilesLoadedMsg struct {
	Path  string
	Files []string
	Error error
}

// LoadFilesCmd loads tar files and directories from a path for browsing
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

		// Separate directories and .tar files
		var dirs []string
		var tarFiles []string

		for _, entry := range entries {
			if entry.IsDir() {
				// Add directory with trailing slash
				dirs = append(dirs, filepath.Join(dirPath, entry.Name())+"/")
			} else if strings.HasSuffix(entry.Name(), ".tar") {
				tarFiles = append(tarFiles, filepath.Join(dirPath, entry.Name()))
			}
		}

		// Add directories first, then .tar files
		files = append(files, dirs...)
		files = append(files, tarFiles...)

		return FilesLoadedMsg{
			Path:  dirPath,
			Files: files,
			Error: nil,
		}
	}
}

// CreateContextCmd creates a new Docker context
func CreateContextCmd(name, dockerHost, tlsPath string, useTLS bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if useTLS && tlsPath != "" {
			err = docker.CreateContextWithTLS(name, dockerHost, tlsPath, false)
		} else {
			err = docker.CreateContext(name, dockerHost)
		}
		return ContextCreatedMsg{
			ContextName: name,
			Success:     err == nil,
			Error:       err,
		}
	}
}

// CreateContextWithCertFilesCmd creates a new Docker context with individual cert files
func CreateContextWithCertFilesCmd(name, description, dockerHost, caFile, certFile, keyFile string, useTLS bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if useTLS && caFile != "" && certFile != "" && keyFile != "" {
			err = docker.CreateContextWithCertFiles(name, description, dockerHost, caFile, certFile, keyFile, false)
		} else {
			err = docker.CreateContext(name, dockerHost)
		}
		return ContextCreatedMsg{
			ContextName: name,
			Success:     err == nil,
			Error:       err,
		}
	}
}

// UpdateContextDescriptionCmd updates only the description of an existing Docker context
func UpdateContextDescriptionCmd(name, description string) tea.Cmd {
	return func() tea.Msg {
		err := docker.UpdateContextDescription(name, description)
		return ContextUpdatedMsg{
			ContextName: name,
			Success:     err == nil,
			Error:       err,
		}
	}
}

// UpdateContextWithCertFilesCmd updates an existing Docker context with individual cert files
func UpdateContextWithCertFilesCmd(name, description, dockerHost, caFile, certFile, keyFile string, useTLS bool) tea.Cmd {
	return func() tea.Msg {
		var err error
		if useTLS && caFile != "" && certFile != "" && keyFile != "" {
			err = docker.UpdateContextWithCertFiles(name, description, dockerHost, caFile, certFile, keyFile, false)
		} else {
			// Update without TLS
			err = docker.UpdateContextWithCertFiles(name, description, dockerHost, "", "", "", false)
		}
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
