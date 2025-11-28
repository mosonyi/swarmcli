package main

import (
	"swarmcli/app"
	swarmlog "swarmcli/utils/log"

	tea "github.com/charmbracelet/bubbletea"
)

// Version information, set by GoReleaser at build time
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
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
