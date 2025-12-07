package command

import (
	"swarmcli/args"
	"swarmcli/registry"
	contextsview "swarmcli/views/contexts"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

type Contexts struct{}

func (Contexts) Name() string        { return "contexts" }
func (Contexts) Description() string { return "List and switch Docker contexts" }

func (Contexts) Execute(ctx any, args args.Args) tea.Cmd {
	return func() tea.Msg {
		return view.NavigateToMsg{
			ViewName: contextsview.ViewName,
			Payload:  nil,
		}
	}
}

var contextsCmd = Contexts{}

func init() {
	registry.Register(contextsCmd)
	// Register aliases
	registry.Register(aliasCommand{name: "context", target: contextsCmd})
	registry.Register(aliasCommand{name: "ctx", target: contextsCmd})
}
