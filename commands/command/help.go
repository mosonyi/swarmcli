package command

import (
	"swarmcli/commands/api"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

type Help struct{}

func (Help) Name() string        { return "help" }
func (Help) Description() string { return "Show all available commands" }

func (Help) Execute(ctx api.Context, args []string) tea.Cmd {
	return func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: "help",
			Payload:  nil,
		}
	}
}
