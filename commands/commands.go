package commands

import tea "github.com/charmbracelet/bubbletea"

type Context struct {
	App any // you can later put references to models or views here
}

type Command interface {
	Name() string        // e.g. "stack ls"
	Description() string // for autocomplete / help
	Execute(ctx Context, args []string) tea.Cmd
}
