package main

import (
	"swarmcli/app"
	swarmlog "swarmcli/utils/log"

	tea "github.com/charmbracelet/bubbletea"
)

// Version information, set by GoReleaser at build time
var (
	version = "v1.1.0"
	commit  = "none"
	date    = "unknown"
)

func init() {
	app.SetVersion(version)
	app.Init()
	// Log version info for debugging
	swarmlog.L().Infof("swarmcli version=%s commit=%s date=%s", version, commit, date)
}

func main() {
	p := tea.NewProgram(app.InitialModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		swarmlog.L().Fatal(err)
	}
}
