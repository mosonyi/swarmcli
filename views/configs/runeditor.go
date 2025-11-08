package configsview

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"swarmcli/docker"
)

func runEditorForConfig(name string) (*docker.ConfigWithDecodedData, error) {
	ctx := context.Background()
	cfg, err := docker.InspectConfig(ctx, name)
	if err != nil {
		return nil, err
	}

	tmp, err := os.CreateTemp("", fmt.Sprintf("%s-*.conf", cfg.Config.Spec.Name))
	if err != nil {
		return nil, fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(cfg.Data); err != nil {
		return nil, fmt.Errorf("failed to write config to temp file: %w", err)
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
		return nil, fmt.Errorf("editor failed: %w", err)
	}

	newData, err := os.ReadFile(tmp.Name())
	if err != nil {
		return nil, fmt.Errorf("failed to read modified file: %w", err)
	}

	if string(newData) == string(cfg.Data) {
		return nil, nil // no changes
	}

	newCfg, err := docker.CreateConfigVersion(ctx, cfg.Config, newData)
	if err != nil {
		return nil, err
	}

	wrapped := docker.ConfigWithDecodedData{
		Config: newCfg,
		Data:   newData,
	}
	return &wrapped, nil
}
