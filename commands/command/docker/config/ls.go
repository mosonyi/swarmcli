package config

import (
	"swarmcli/args"
	"swarmcli/registry"
	configsview "swarmcli/views/configs"

	tea "github.com/charmbracelet/bubbletea"
)
import "swarmcli/views/view"

type DockerConfigLs struct{}

func (DockerConfigLs) Name() string        { return "config" }
func (DockerConfigLs) Description() string { return "docker config ls" }

func (DockerConfigLs) Execute(ctx any, args args.Args) tea.Cmd {
	return func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: configsview.ViewName,
			Payload:  nil,
		}
	}
}

var lsCmd = DockerConfigLs{}

func init() {
	registry.Register(lsCmd)
}
