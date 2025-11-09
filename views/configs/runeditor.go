package configsview

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"swarmcli/docker"

	tea "github.com/charmbracelet/bubbletea"
)

// editConfigInEditorCmd runs the external editor using ExecProcess with detailed logging
func editConfigInEditorCmd(name string) tea.Cmd {
	l().Infoln("editConfigInEditorCmd: started")

	ctx := context.Background()
	cfg, err := docker.InspectConfig(ctx, name)
	if err != nil {
		l().Infoln("InspectConfig error:", err)
		return func() tea.Msg { return editConfigErrorMsg{err} }
	}
	l().Infoln("InspectConfig OK")

	tmp, err := os.CreateTemp("", fmt.Sprintf("%s-*.conf", cfg.Config.Spec.Name))
	if err != nil {
		l().Infoln("CreateTemp error:", err)
		return func() tea.Msg { return editConfigErrorMsg{fmt.Errorf("failed to create temp file: %w", err)} }
	}
	l().Infoln("Created tmp file:", tmp.Name())
	defer os.Remove(tmp.Name())

	if _, err := tmp.Write(cfg.Data); err != nil {
		l().Infoln("Write temp error:", err)
		return func() tea.Msg { return editConfigErrorMsg{fmt.Errorf("failed to write config to temp file: %w", err)} }
	}
	tmp.Close()

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "nano"
	}
	l().Infoln("Using editor:", editor)

	cmd := exec.Command(editor, tmp.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	l().Infoln("Prepared command:", cmd.String())

	// RETURN tea.Cmd directly
	l().Infoln("Returning ExecProcess cmd")
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		l().Infoln("In ExecProcess callback")

		if err != nil {
			l().Infoln("Editor returned error:", err)
			return editConfigErrorMsg{fmt.Errorf("editor failed: %w", err)}
		}
		l().Infoln("Editor finished successfully")

		newData, err := os.ReadFile(tmp.Name())
		if err != nil {
			l().Infoln("ReadFile error:", err)
			return editConfigErrorMsg{fmt.Errorf("failed to read edited file: %w", err)}
		}

		if string(newData) == string(cfg.Data) {
			l().Infoln("No changes made to config")
			return editConfigDoneMsg{
				Name:    cfg.Config.Spec.Name,
				Changed: false,
				Config: docker.ConfigWithDecodedData{
					Config: cfg.Config,
					Data:   cfg.Data,
				},
			}
		}

		newCfg, err := docker.CreateConfigVersion(ctx, cfg.Config, newData)
		if err != nil {
			l().Infoln("CreateConfigVersion error:", err)
			return editConfigErrorMsg{err}
		}

		l().Infoln("Config updated:", newCfg.Spec.Name)
		return editConfigDoneMsg{
			Name:    newCfg.Spec.Name,
			Changed: true,
			Config: docker.ConfigWithDecodedData{
				Config: newCfg,
				Data:   newData,
			},
		}
	})
}
