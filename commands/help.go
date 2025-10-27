package commands

import (
	helpview "swarmcli/views/help"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

type HelpCommand struct{}

func (HelpCommand) Name() string        { return "help" }
func (HelpCommand) Description() string { return "Show all available commands" }

func (HelpCommand) Execute(ctx Context, args []string) tea.Cmd {
	return func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: helpview.ViewName,
			Payload:  nil,
		}
	}
}
