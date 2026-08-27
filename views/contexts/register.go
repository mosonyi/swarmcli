// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package contexts

import (
	"github.com/Eldara-Tech/swarmcli/v2/docker"
	"github.com/Eldara-Tech/swarmcli/v2/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	view.RegisterView(ViewName, factory)
}

func factory(deps docker.Deps, w, h int, _ any) (view.View, tea.Cmd) {
	model := New()
	model.deps = deps
	model.Visible = true
	model.SetSize(w, h)
	model.SetLoading(true)
	// No tick: OnEnter arms the one chain, and the app calls it right after
	// this. Arming here instead left the view unable to restart its chain
	// after a drill-down, because OnEnter is all that goBack runs.
	return model, model.loadContextsCmd()
}
