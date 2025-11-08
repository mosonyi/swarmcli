package configsview

import (
	"context"
	"fmt"
	"swarmcli/docker"

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

func editConfigCmd(name string) tea.Cmd {
	return func() tea.Msg {
		return editConfigMsg{Name: name}
	}
}

func rotateConfigCmd(name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		cfg, err := docker.InspectConfig(ctx, name)
		if err != nil {
			return errorMsg(err)
		}

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
		return nil
		//return view.SwitchTo("inspect", name)
	}
}
