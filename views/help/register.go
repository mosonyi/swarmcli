// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package helpview

import (
	"swarmcli/docker"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	view.RegisterView(ViewName, factory)
}

func factory(_ docker.Deps, w, h int, payload any) (view.View, tea.Cmd) {
	if categories, ok := payload.([]HelpCategory); ok {
		return NewDetailed(w, h, categories), nil
	}
	cmds, _ := payload.([]CommandInfo)
	return New(w, h, cmds), nil
}
