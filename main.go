package main

import (
	"swarmcli/app"
	swarmlog "swarmcli/utils/log"

	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	app.Init()
}

func main() {
	p := tea.NewProgram(app.InitialModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		swarmlog.L().Fatal(err)
	}
}
