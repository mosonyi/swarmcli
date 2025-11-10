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
