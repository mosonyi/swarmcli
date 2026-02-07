// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package main

import (
	"runtime/debug"
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
	app.SetVersion(version)
	app.Init()
	// Log version info for debugging
	swarmlog.L().Infof("swarmcli version=%s commit=%s date=%s", version, commit, date)
}

func main() {
	p := tea.NewProgram(app.InitialModel(), tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		// Log full stack trace to aid debugging panic-causes inside the TUI
		swarmlog.L().Errorf("program exited with error: %v", err)
		swarmlog.L().Errorf("stack trace:\n%s", string(debug.Stack()))
		swarmlog.L().Fatal(err)
	}
}
