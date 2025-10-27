package api

import tea "github.com/charmbracelet/bubbletea"

type Context struct {
	App any // e.g. *app.Model
}

type Command interface {
	Name() string
	Description() string
	Execute(ctx Context, args []string) tea.Cmd
}
