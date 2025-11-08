package configsview

import (
	"context"
	"fmt"
	"os"
	"os/exec"
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
		ctx := context.Background()
		cfg, err := docker.InspectConfig(ctx, name)
		if err != nil {
			return errorMsg(err)
		}

		// Create a temp file for editing
		tmp, err := os.CreateTemp("", fmt.Sprintf("%s-*.conf", cfg.Config.Spec.Name))
		if err != nil {
			return errorMsg(fmt.Errorf("failed to create temp file: %w", err))
		}
		defer os.Remove(tmp.Name())

		if _, err := tmp.Write(cfg.Data); err != nil {
			return errorMsg(fmt.Errorf("failed to write config to temp file: %w", err))
		}
		tmp.Close()

		editor := os.Getenv("EDITOR")
		if editor == "" {
			editor = "nano"
		}

		cmd := exec.Command(editor, tmp.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		cleanup := tea.Suspend()
		defer cleanup()

		if err := cmd.Run(); err != nil {
			return errorMsg(fmt.Errorf("editor failed: %w", err))
		}

		// Read modified data
		newData, err := os.ReadFile(tmp.Name())
		if err != nil {
			return errorMsg(fmt.Errorf("failed to read modified file: %w", err))
		}

		if string(newData) == string(cfg.Data) {
			return tea.Printf("No changes made to %s", cfg.Config.Spec.Name)
		}

		newCfg, err := docker.CreateConfigVersion(ctx, cfg.Config, newData)
		if err != nil {
			return errorMsg(err)
		}

		return configUpdatedMsg{
			Old: *cfg,
			New: docker.ConfigWithDecodedData{Config: newCfg, Data: newData},
		}
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
