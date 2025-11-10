package configsview

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sort"
	"strings"
	"swarmcli/docker"

	tea "github.com/charmbracelet/bubbletea"
)

// editConfigInEditorCmd launches the editor with human-readable JSON for a config
func editConfigInEditorCmd(name string) tea.Cmd {
	l().Infoln("editConfigInEditorCmd: started")

	ctx := context.Background()
	cfg, err := docker.InspectConfig(ctx, name)
	if err != nil {
		l().Infoln("InspectConfig error:", err)
		return func() tea.Msg { return editConfigErrorMsg{err} }
	}
	l().Infoln("InspectConfig OK")

	// Serialize config to pretty JSON for human-readable editing
	content, err := cfg.PrettyJSON()
	if err != nil {
		l().Infoln("PrettyJSON error:", err)
		return func() tea.Msg { return editConfigErrorMsg{fmt.Errorf("failed to marshal config: %w", err)} }
	}

	tmp, err := os.CreateTemp("", fmt.Sprintf("%s-*.json", cfg.Config.Spec.Name))
	if err != nil {
		l().Infoln("CreateTemp error:", err)
		return func() tea.Msg { return editConfigErrorMsg{fmt.Errorf("failed to create temp file: %w", err)} }
	}
	l().Infoln("Created tmp file:", tmp.Name())

	if _, err := tmp.Write(content); err != nil {
		l().Infoln("Write temp error:", err)
		return func() tea.Msg { return editConfigErrorMsg{fmt.Errorf("failed to write temp file: %w", err)} }
	}
	tmp.Close()
	l().Infoln("Wrote JSON to temp file, length:", len(content))

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
	l().Infoln("Returning ExecProcess cmd")

	// Run editor via Bubble Tea ExecProcess
	return tea.ExecProcess(cmd, func(err error) tea.Msg {
		defer os.Remove(tmp.Name())

		l().Infoln("In ExecProcess callback")
		if err != nil {
			l().Infoln("Editor returned error:", err)
			return editConfigErrorMsg{fmt.Errorf("editor failed: %w", err)}
		}
		l().Infoln("Editor finished successfully")

		// Read edited JSON
		editedJSON, err := os.ReadFile(tmp.Name())
		if err != nil {
			l().Infoln("ReadFile error:", err)
			return editConfigErrorMsg{fmt.Errorf("failed to read edited file: %w", err)}
		}

		// Parse edited JSON back into a struct with flexible DataParsed type
		var tmpStruct struct {
			Config     docker.ConfigWithDecodedData `json:"Config"`
			DataParsed any                          `json:"DataParsed,omitempty"`
			RawData    string                       `json:"DataRaw,omitempty"`
		}
		if err := json.Unmarshal(editedJSON, &tmpStruct); err != nil {
			l().Infoln("JSON unmarshal error:", err)
			return editConfigErrorMsg{fmt.Errorf("failed to parse edited JSON: %w", err)}
		}

		// Rebuild the config Data based on the type of DataParsed
		var newData []byte
		switch v := tmpStruct.DataParsed.(type) {
		case map[string]any:
			// Preserve key order for consistent base64 and editing
			keys := make([]string, 0, len(v))
			for k := range v {
				keys = append(keys, k)
			}
			sort.Strings(keys)

			lines := make([]string, 0, len(v))
			for _, k := range keys {
				lines = append(lines, fmt.Sprintf("%s=%v", k, v[k]))
			}
			newData = []byte(strings.Join(lines, "\n"))

		case string:
			newData = []byte(v) // use raw string

		default:
			newData = tmpStruct.Config.Data // fallback to original data
		}
		l().Infoln("Rebuilt newData length:", len(newData))

		// Check if nothing changed
		if string(newData) == string(cfg.Data) {
			l().Infoln("No changes made to config")
			return editConfigDoneMsg{
				Name:      cfg.Config.Spec.Name,
				Changed:   false,
				OldConfig: *cfg,
				NewConfig: *cfg,
			}
		}

		// Create a new Docker config version
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
