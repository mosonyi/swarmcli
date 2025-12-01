package configsview

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"swarmcli/docker"

	tea "github.com/charmbracelet/bubbletea"
)

// editConfigInEditorCmd creates a tmp file and opens the editor to edit the new config.
func editConfigInEditorCmd(name string) tea.Cmd {
	l().Infoln("editConfigInEditorCmd: started")

	ctx := context.Background()
	cfg, err := docker.InspectConfig(ctx, name)
	if err != nil {
		l().Infoln("InspectConfig error:", err)
		return func() tea.Msg { return editConfigErrorMsg{err} }
	}
	l().Infoln("InspectConfig OK")

	// Get the human-readable UTF-8 content to edit
	content := cfg.Data
	l().Infoln("Prepared editable content, length:", len(content))

	tmp, err := os.CreateTemp("", fmt.Sprintf("%s-*.txt", cfg.Config.Spec.Name))
	if err != nil {
		l().Infoln("CreateTemp error:", err)
		return func() tea.Msg { return editConfigErrorMsg{fmt.Errorf("failed to create temp file: %w", err)} }
	}
	defer func(tmp *os.File) {
		err := tmp.Close()
		if err != nil {
			l().Errorln("Failed to close temp file:", tmp.Name(), "error:", err)
		}
	}(tmp)

	if _, err := tmp.Write(content); err != nil {
		l().Infoln("Write temp error:", err)
		return func() tea.Msg { return editConfigErrorMsg{fmt.Errorf("failed to write temp file: %w", err)} }
	}
	l().Infoln("Wrote plain data to temp file:", tmp.Name())

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

	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		defer func(name string) {
			err := os.Remove(name)
			if err != nil {
				l().Errorln("Failed to remove temp file:", name, "error:", err)
			}
		}(tmp.Name())

		l().Infoln("In ExecProcess callback")
		if err != nil {
			l().Infoln("Editor returned error:", err)
			return editConfigErrorMsg{fmt.Errorf("editor failed: %w", err)}
		}
		l().Infoln("Editor finished successfully")

		// Read the edited plain UTF-8 data
		newData, err := os.ReadFile(tmp.Name())
		if err != nil {
			l().Infoln("ReadFile error:", err)
			return editConfigErrorMsg{fmt.Errorf("failed to read edited file: %w", err)}
		}
		l().Infoln("Read edited data, length:", len(newData))

		// Check if content changed
		if string(newData) == string(cfg.Data) {
			l().Infoln("No changes made to config")
			return editConfigDoneMsg{
				Name:      cfg.Config.Spec.Name,
				Changed:   false,
				OldConfig: *cfg,
				NewConfig: *cfg,
			}
		}

		// Create a new Docker config version with the edited data
		newCfg, err := docker.CreateConfigVersion(ctx, cfg.Config, newData)
		if err != nil {
			l().Infoln("CreateConfigVersion error:", err)
			return editConfigErrorMsg{err}
		}
		l().Infoln("Config updated:", newCfg.Spec.Name)

		wrapped := docker.ConfigWithDecodedData{
			Config: newCfg,
			Data:   newData,
		}

		return editConfigDoneMsg{
			Name:      newCfg.Spec.Name,
			Changed:   true,
			OldConfig: *cfg,
			NewConfig: wrapped,
		}
	})
}
