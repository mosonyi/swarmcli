package command

import (
	"swarmcli/commands"
	"swarmcli/commands/api"
	helpview "swarmcli/views/help"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

type Help struct{}

func (Help) Name() string        { return "help" }
func (Help) Description() string { return "Show all available commands" }

func (Help) Execute(ctx api.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: helpview.ViewName,
			Payload:  AllCommandInfos(),
		}
	}
}

func AllCommandInfos() []helpview.CommandInfo {
	cmds := []helpview.CommandInfo{}
	for _, cmd := range commands.All() {
		cmds = append(cmds, helpview.CommandInfo{
			Name:        cmd.Name(),
			Description: cmd.Description(),
		})
	}
	return cmds
}
