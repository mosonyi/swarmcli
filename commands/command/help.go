package command

import (
	"swarmcli/args"
	"swarmcli/registry"
	helpview "swarmcli/views/help"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

type Help struct{}

func (Help) Name() string        { return "help" }
func (Help) Description() string { return "Show all available commands" }

func (Help) Execute(ctx any, args args.Args) tea.Cmd {
	return func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: helpview.ViewName,
			Payload:  AllCommandInfos(),
		}
	}
}

func AllCommandInfos() []helpview.CommandInfo {
	var cmds []helpview.CommandInfo
	// Go technicality. Need to call `registry` directly.
	// We can't depend on the parent package, as it creates
	// a cycle.
	for _, cmd := range registry.All() {
		cmds = append(cmds, helpview.CommandInfo{
			Name:        cmd.Name(),
			Description: cmd.Description(),
		})
	}
	return cmds
}

var helpCmd = Help{}

func init() {
	registry.Register(helpCmd)
}
