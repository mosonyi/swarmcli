// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package command

import (
	"os"
	"strings"

	"swarmcli/args"
	"swarmcli/registry"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

// DevUpdate is the dev-only :dev-update command. It force-shows the startup
// "update available" notice on demand for previewing, bypassing the per-version
// dismissal. Registered only under SWARMCLI_ENV=dev (see init), so it does not
// exist in production builds.
type DevUpdate struct{}

func (DevUpdate) Name() string        { return "dev-update" }
func (DevUpdate) Description() string { return "Dev only: preview the update-available notice" }

func (DevUpdate) Spec() registry.CommandSpec {
	return registry.CommandSpec{
		Usage: "[version]",
		Detail: "Dev-only preview of the startup \"update available\" notice. " +
			"Force-shows the dialog on demand (optionally for a given version), " +
			"bypassing the per-version dismissal. Registered only under " +
			"SWARMCLI_ENV=dev.",
		Examples: []string{":dev-update", ":dev-update v9.9.9"},
	}
}

func (DevUpdate) Execute(_ any, a args.Args) tea.Cmd {
	var ver string
	if len(a.Positionals) > 0 {
		ver = strings.TrimSpace(a.Positionals[0])
	}
	return func() tea.Msg {
		return view.OpenUpdateDialogMsg{Version: ver}
	}
}

// devUpdateEnabled reports whether the dev-only command should register.
func devUpdateEnabled() bool { return os.Getenv("SWARMCLI_ENV") == "dev" }

func init() {
	if devUpdateEnabled() {
		registry.Register(DevUpdate{})
	}
}
