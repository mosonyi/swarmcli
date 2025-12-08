package configsview

import (
	"context"
	"fmt"
	"os"
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

func loadConfigsCmd() tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		cfgs, err := docker.ListConfigs(ctx)
		if err != nil {
			return errorMsg(fmt.Errorf("failed to list configs: %w", err))
		}

		wrapped := make([]docker.ConfigWithDecodedData, len(cfgs))
		for i, c := range cfgs {
			wrapped[i] = docker.ConfigWithDecodedData{Config: c, Data: c.Spec.Data}
		}
		return configsLoadedMsg(wrapped)
	}
}

// CheckConfigsCmd checks if configs have changed and returns update message if so
func CheckConfigsCmd(lastHash uint64) tea.Cmd {
	return func() tea.Msg {
		l().Info("CheckConfigsCmd: Polling for config changes")

		ctx := context.Background()
		cfgs, err := docker.ListConfigs(ctx)
		if err != nil {
			l().Errorf("CheckConfigsCmd: ListConfigs failed: %v", err)
			return tickCmd()
		}

		wrapped := make([]docker.ConfigWithDecodedData, len(cfgs))
		for i, c := range cfgs {
			wrapped[i] = docker.ConfigWithDecodedData{Config: c, Data: c.Spec.Data}
		}

		// Create a stable hash based only on ID and Version (not timestamps)
		type stableConfig struct {
			ID      string
			Version uint64
			Name    string
		}
		stableConfigs := make([]stableConfig, len(cfgs))
		for i, c := range cfgs {
			stableConfigs[i] = stableConfig{
				ID:      c.ID,
				Version: c.Version.Index,
				Name:    c.Spec.Name,
			}
		}

		newHash, err := hash.Compute(stableConfigs)
		if err != nil {
			l().Errorf("CheckConfigsCmd: Error computing hash: %v", err)
			// Schedule next poll even on error
			return tickCmd()
		}

		l().Infof("CheckConfigsCmd: lastHash=%s, newHash=%s, configCount=%d",
			hash.Fmt(lastHash), hash.Fmt(newHash), len(wrapped))

		// Only return update message if something changed
		if newHash != lastHash {
			l().Info("CheckConfigsCmd: Change detected! Refreshing config list")
			return configsLoadedMsg(wrapped)
		}

		l().Info("CheckConfigsCmd: No changes detected, scheduling next poll")
		// Schedule next poll in 5 seconds
		return tickCmd()
	}
}

func rotateConfigCmd(oldCfg *docker.ConfigWithDecodedData, newCfg *docker.ConfigWithDecodedData) tea.Cmd {
	if newCfg == nil {
		return nil
	}

	l().Debugln("Starting to rotate config", newCfg.Config.Spec.Name)
	return func() tea.Msg {
		ctx := context.Background()

		oldSwarmCfg := &swarm.Config{}
		if oldCfg != nil {
			oldSwarmCfg = &oldCfg.Config
		}

		if err := docker.RotateConfigInServices(ctx, oldSwarmCfg, newCfg.Config); err != nil {
			return errorMsg(err)
		}

		result := configRotatedMsg{
			New: *newCfg,
		}
		if oldCfg != nil {
			result.Old = *oldCfg
		}
		return result
	}
}

func inspectConfigCmd(name string) tea.Cmd {
	return func() tea.Msg {
		cfg, err := docker.InspectConfig(context.Background(), name)
		jsonStr := ""
		if err != nil {
			jsonStr = fmt.Sprintf("Error inspecting config %q: %v", name, err)
		} else if data, err := cfg.JSON(); err != nil {
			jsonStr = fmt.Sprintf("Error marshalling config %q: %v", name, err)
		} else {
			jsonStr = string(data)
		}

		return view.NavigateToMsg{
			ViewName: inspectview.ViewName,
			Payload: map[string]interface{}{
				"title": fmt.Sprintf("Config: %s", name),
				"json":  jsonStr,
				"meta": map[string]interface{}{
					"ID":   cfg.Config.ID,
					"Name": cfg.Config.Spec.Name,
					"Data": len(cfg.Config.Spec.Data),
				},
			},
		}
	}
}

func inspectRawConfigCmd(name string) tea.Cmd {
	return func() tea.Msg {
		cfg, err := docker.InspectConfig(context.Background(), name)
		if err != nil {
			return view.NavigateToMsg{
				ViewName: inspectview.ViewName,
				Payload: map[string]interface{}{
					"title": fmt.Sprintf("Config: %s", name),
					"json":  fmt.Sprintf("Error loading config %q: %v", name, err),
				},
			}
		}

		// Use *plain content*, same as editor:
		raw := string(cfg.Data)

		return view.NavigateToMsg{
			ViewName: inspectview.ViewName,
			Payload: map[string]interface{}{
				"title":  fmt.Sprintf("Config (raw): %s", name),
				"json":   raw,
				"format": inspectview.FormatRaw,
			},
		}
	}
}

func deleteConfigCmd(name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		err := docker.DeleteConfig(ctx, name)
		if err != nil {
			return errorMsg(fmt.Errorf("failed to delete config %q: %w", name, err))
		}
		return configDeletedMsg{Name: name}
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
			return filesLoadedMsg{
				Path:  dirPath,
				Files: files,
				Error: err,
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
				// Add all files (not just .tar like in contexts)
				regFiles = append(regFiles, filepath.Join(dirPath, entry.Name()))
			}
		}

		// Add directories first, then files
		files = append(files, dirs...)
		files = append(files, regFiles...)

		return filesLoadedMsg{
			Path:  dirPath,
			Files: files,
			Error: nil,
		}
	}
}

func createConfigFromFileCmd(name, filePath string) tea.Cmd {
	return func() tea.Msg {
		l().Infof("Creating config %s from file %s", name, filePath)

		// Read file content
		data, err := os.ReadFile(filePath)
		if err != nil {
			l().Errorf("Failed to read file %s: %v", filePath, err)
			return configCreateErrorMsg{fmt.Errorf("failed to read file: %w", err)}
		}

		// Create the config
		ctx := context.Background()
		newCfg, err := docker.CreateConfig(ctx, name, data)
		if err != nil {
			l().Errorf("Failed to create config %s: %v", name, err)
			// Return error with file path so we can retry with corrected name
			return fileContentReadyMsg{Name: name, FilePath: filePath, Data: data, Err: err}
		}

		l().Infof("Successfully created config %s from file", name)
		return configCreatedMsg{Config: newCfg}
	}
}

func getUsedByStacksCmd(configName string) tea.Cmd {
	return func() tea.Msg {
		l().Infof("Getting stacks that use config: %s", configName)

		ctx := context.Background()
		services, err := docker.ListServicesUsingConfigName(ctx, configName)
		if err != nil {
			l().Errorf("Failed to list services using config %s: %v", configName, err)
			return usedByMsg{ConfigName: configName, Stacks: nil, Error: err}
		}

		// Extract unique stack names
		stackSet := make(map[string]bool)
		for _, svc := range services {
			if stackName, ok := svc.Spec.Labels["com.docker.stack.namespace"]; ok && stackName != "" {
				stackSet[stackName] = true
			}
		}

		// Convert to sorted slice
		stacks := make([]string, 0, len(stackSet))
		for stack := range stackSet {
			stacks = append(stacks, stack)
		}

		// Sort alphabetically
		sort.Strings(stacks)

		l().Infof("Config %s is used by %d stack(s)", configName, len(stacks))
		return usedByMsg{ConfigName: configName, Stacks: stacks, Error: nil}
	}
}
