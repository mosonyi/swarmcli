package configsview

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"swarmcli/docker"

	tea "github.com/charmbracelet/bubbletea"
)

// Runs the external editor and returns a message when done
func editConfigInEditorCmd(name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		cfg, err := docker.InspectConfig(ctx, name)
		if err != nil {
			return err
		}

		tmp, err := os.CreateTemp("", fmt.Sprintf("%s-*.conf", cfg.Config.Spec.Name))
		if err != nil {
			return fmt.Errorf("failed to create temp file: %w", err)
		}
		defer os.Remove(tmp.Name())

		if _, err := tmp.Write(cfg.Data); err != nil {
			return fmt.Errorf("failed to write config to temp file: %w", err)
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

		if err := cmd.Run(); err != nil {
			return fmt.Errorf("editor failed: %w", err)
		}

		newData, err := os.ReadFile(tmp.Name())
		if err != nil {
			return fmt.Errorf("failed to read modified file: %w", err)
		}

		if string(newData) == string(cfg.Data) {
			return editConfigDoneMsg{cfg.Config.Spec.Name, false, docker.ConfigWithDecodedData{Config: cfg.Config, Data: cfg.Data}} // no changes
		}

		newCfg, err := docker.CreateConfigVersion(ctx, cfg.Config, newData)
		if err != nil {
			return err
		}

		wrapped := docker.ConfigWithDecodedData{
			Config: newCfg,
			Data:   newData,
		}

		return editConfigDoneMsg{
			newCfg.Spec.Name,
			true,
			wrapped,
		}
	}
}
