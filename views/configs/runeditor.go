package configsview

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"swarmcli/docker"

	tea "github.com/charmbracelet/bubbletea"
)

// editConfigInEditorCmd runs the external editor using ExecProcess
func editConfigInEditorCmd(name string) tea.Cmd {
	return func() tea.Msg {
		ctx := context.Background()
		cfg, err := docker.InspectConfig(ctx, name)
		if err != nil {
			return editConfigErrorMsg{err}
		}

		l().Infoln("Creating tmp dir")
		tmp, err := os.CreateTemp("", fmt.Sprintf("%s-*.conf", cfg.Config.Spec.Name))
		if err != nil {
			return editConfigErrorMsg{fmt.Errorf("failed to create temp file: %w", err)}
		}
		defer os.Remove(tmp.Name())

		if _, err := tmp.Write(cfg.Data); err != nil {
			return editConfigErrorMsg{fmt.Errorf("failed to write config to temp file: %w", err)}
		}
		tmp.Close()
		l().Infoln("Created tmp file")

		editor := os.Getenv("EDITOR")

		l().Infoln("Opening editor:", editor)
		if editor == "" {
			editor = "nano"
		}

		l().Infoln("Executing command")

		cmd := exec.Command(editor, tmp.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		// THIS IS THE FIX: return the tea.Cmd directly
		return tea.ExecProcess(cmd, func(err error) tea.Msg {
			l().Infoln("In t.ExecProcess")
			if err != nil {
				return editConfigErrorMsg{fmt.Errorf("editor failed: %w", err)}
			}

			newData, err := os.ReadFile(tmp.Name())
			if err != nil {
				return editConfigErrorMsg{fmt.Errorf("failed to read edited file: %w", err)}
			}

			if string(newData) == string(cfg.Data) {
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
				return editConfigErrorMsg{err}
			}

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
}
