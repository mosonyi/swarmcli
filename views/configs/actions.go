// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

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

func (m *Model) loadConfigsCmd() tea.Cmd {
	configOps := m.deps.Configs
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), pollTimeout)
		defer cancel()
		cfgs, err := configOps.ListConfigs(ctx)
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

// computeConfigUsedCmd checks which configs are used by services in background
// and returns a usedStatusUpdatedMsg containing a map[id]bool.
func (m *Model) computeConfigUsedCmd(cfgs []docker.ConfigWithDecodedData) tea.Cmd {
	configOps := m.deps.Configs
	return func() tea.Msg {
		usedMap := make(map[string]bool, len(cfgs))
		ctx, cancel := context.WithTimeout(context.Background(), pollTimeout)
		defer cancel()
		for _, c := range cfgs {
			usedMap[c.Config.ID] = false
			svcs, err := configOps.ListServicesUsingConfigID(ctx, c.Config.ID)
			if err == nil && len(svcs) > 0 {
				usedMap[c.Config.ID] = true
			}
		}
		return usedStatusUpdatedMsg(usedMap)
	}
}

// checkConfigsCmd checks if configs have changed and returns update message if so
func (m *Model) checkConfigsCmd(lastHash uint64) tea.Cmd {
	configOps := m.deps.Configs
	return func() tea.Msg {
		if !m.polling.CompareAndSwap(false, true) {
			l().Info("checkConfigsCmd: skipped, previous poll still in flight")
			return PollRetryMsg{}
		}
		defer m.polling.Store(false)

		l().Info("checkConfigsCmd: Polling for config changes")

		ctx, cancel := context.WithTimeout(context.Background(), pollTimeout)
		defer cancel()
		cfgs, err := configOps.ListConfigs(ctx)
		if err != nil {
			l().Errorf("checkConfigsCmd: ListConfigs failed: %v", err)
			return PollRetryMsg{}
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
			l().Errorf("checkConfigsCmd: Error computing hash: %v", err)
			// Schedule next poll even on error
			return PollRetryMsg{}
		}

		l().Infof("checkConfigsCmd: lastHash=%s, newHash=%s, configCount=%d",
			hash.Fmt(lastHash), hash.Fmt(newHash), len(wrapped))

		// Only return update message if something changed
		if newHash != lastHash {
			l().Info("checkConfigsCmd: Change detected! Refreshing config list")
			return configsLoadedMsg(wrapped)
		}

		l().Info("checkConfigsCmd: No changes detected, scheduling next poll")
		return PollRetryMsg{}
	}
}

func (m *Model) rotateConfigCmd(oldCfg *docker.ConfigWithDecodedData, newCfg *docker.ConfigWithDecodedData) tea.Cmd {
	if newCfg == nil {
		return nil
	}

	configOps := m.deps.Configs
	l().Debugln("Starting to rotate config", newCfg.Config.Spec.Name)
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), userActionTimeout)
		defer cancel()

		oldSwarmCfg := &swarm.Config{}
		if oldCfg != nil {
			oldSwarmCfg = &oldCfg.Config
		}

		if err := configOps.RotateConfigInServices(ctx, oldSwarmCfg, newCfg.Config); err != nil {
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

func (m *Model) inspectConfigCmd(name string) tea.Cmd {
	configOps := m.deps.Configs
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), userActionTimeout)
		defer cancel()
		cfg, err := configOps.InspectConfig(ctx, name)
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

func (m *Model) inspectRawConfigCmd(name string) tea.Cmd {
	configOps := m.deps.Configs
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), userActionTimeout)
		defer cancel()
		cfg, err := configOps.InspectConfig(ctx, name)
		if err != nil {
			return view.NavigateToMsg{
				ViewName: inspectview.ViewName,
				Payload: map[string]interface{}{
					"title": fmt.Sprintf("Config: %s", name),
					"json":  fmt.Sprintf("Error loading config %q: %v", name, err),
				},
			}
		}

		// Use *plain content*, same as editor (decompressing gzip payloads such
		// as chart release records so the raw view shows text, not binary):
		raw := string(cfg.DisplayData())

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

func (m *Model) deleteConfigCmd(name string) tea.Cmd {
	configOps := m.deps.Configs
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), userActionTimeout)
		defer cancel()
		err := configOps.DeleteConfig(ctx, name)
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

func (m *Model) createConfigFromFileCmd(name, filePath string, labels map[string]string) tea.Cmd {
	configOps := m.deps.Configs
	return func() tea.Msg {
		l().Infof("Creating config %s from file %s (labels=%v)", name, filePath, labels)

		// Read file content
		data, err := os.ReadFile(filePath)
		if err != nil {
			l().Errorf("Failed to read file %s: %v", filePath, err)
			return configCreateErrorMsg{fmt.Errorf("failed to read file: %w", err)}
		}

		// Create the config
		ctx, cancel := context.WithTimeout(context.Background(), userActionTimeout)
		defer cancel()
		newCfg, err := configOps.CreateConfig(ctx, name, data, labels)
		if err != nil {
			l().Errorf("Failed to create config %s: %v", name, err)
			// Return error with file path so we can retry with corrected name
			return fileContentReadyMsg{Name: name, FilePath: filePath, Data: data, Err: err}
		}

		l().Infof("Successfully created config %s from file", name)
		return configCreatedMsg{Config: newCfg}
	}
}

func (m *Model) createConfigFromContentCmd(name string, content []byte, labels map[string]string) tea.Cmd {
	configOps := m.deps.Configs
	return func() tea.Msg {
		l().Infof("Creating config %s from inline content (labels=%v)", name, labels)

		ctx, cancel := context.WithTimeout(context.Background(), userActionTimeout)
		defer cancel()
		newCfg, err := configOps.CreateConfig(ctx, name, content, labels)
		if err != nil {
			l().Errorf("Failed to create config %s: %v", name, err)
			return configCreateErrorMsg{fmt.Errorf("failed to create config: %w", err)}
		}

		l().Infof("Successfully created config %s", name)
		return configCreatedMsg{Config: newCfg}
	}
}

func (m *Model) getUsedByStacksCmd(configName string) tea.Cmd {
	configOps := m.deps.Configs
	return func() tea.Msg {
		l().Infof("Getting stacks/services that use config: %s", configName)

		ctx, cancel := context.WithTimeout(context.Background(), userActionTimeout)
		defer cancel()
		// Get config ID for robust matching
		cfg, err := configOps.InspectConfig(ctx, configName)
		if err != nil {
			l().Errorf("Failed to inspect config %s: %v", configName, err)
			return usedByMsg{ConfigName: configName, UsedBy: nil, Error: err}
		}

		// Get services by config name and ID
		servicesByName, err := configOps.ListServicesUsingConfigName(ctx, configName)
		if err != nil {
			l().Errorf("Failed to list services using config name %s: %v", configName, err)
			return usedByMsg{ConfigName: configName, UsedBy: nil, Error: err}
		}
		servicesByID, err := configOps.ListServicesUsingConfigID(ctx, cfg.Config.ID)
		if err != nil {
			l().Errorf("Failed to list services using config ID %s: %v", cfg.Config.ID, err)
			return usedByMsg{ConfigName: configName, UsedBy: nil, Error: err}
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

		l().Infof("Config %s is used by %d service(s)", configName, len(usedBy))
		return usedByMsg{ConfigName: configName, UsedBy: usedBy, Error: nil}
	}
}
