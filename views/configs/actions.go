package configsview

import (
	"context"
	"fmt"
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

func rotateConfigCmd(cfg *docker.ConfigWithDecodedData) tea.Cmd {
	if cfg == nil {
		return nil
	}

	l().Debugln("Starting to rotate config", cfg.Config.Spec.Name)
	return func() tea.Msg {
		ctx := context.Background()

		// Use the edited config data
		newCfg, err := docker.CreateConfigVersion(ctx, cfg.Config, cfg.Data)
		if err != nil {
			return errorMsg(err)
		}

		if err := docker.RotateConfigInServices(ctx, cfg.Config, newCfg); err != nil {
			return errorMsg(err)
		}

		return configRotatedMsg{
			Old: *cfg,
			New: docker.ConfigWithDecodedData{Config: newCfg, Data: cfg.Data},
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
