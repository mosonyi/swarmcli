package view

import tea "github.com/charmbracelet/bubbletea"

type View interface {
	Update(msg tea.Msg) (View, tea.Cmd)
	View() string
	Init() tea.Cmd
	Name() string
}
