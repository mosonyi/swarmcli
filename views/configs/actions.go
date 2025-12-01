package configsview

import (
	"context"
	"fmt"
	"swarmcli/core/primitives/hash"
	"swarmcli/docker"
	inspectview "swarmcli/views/inspect"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
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

		newHash, err := hash.Compute(wrapped)
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

		if err := docker.RotateConfigInServices(ctx, &oldCfg.Config, newCfg.Config); err != nil {
			return errorMsg(err)
		}

		return configRotatedMsg{
			New: *newCfg,
		}
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
