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

// SwitchContextCmd switches to a different Docker context
func SwitchContextCmd(contextName string) tea.Cmd {
	return func() tea.Msg {
		err := docker.UseContext(contextName)
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

// LoadFilesCmd loads tar files from a directory
func LoadFilesCmd(dirPath string) tea.Cmd {
	return func() tea.Msg {
		files := []string{}
		entries, err := os.ReadDir(dirPath)
		if err != nil {
			return FilesLoadedMsg{
				Path:  dirPath,
				Files: files,
				Error: err,
			}
		}
		for _, entry := range entries {
			if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".tar") {
				files = append(files, filepath.Join(dirPath, entry.Name()))
			}
		}
		return FilesLoadedMsg{
			Path:  dirPath,
			Files: files,
			Error: nil,
		}
	}
}
