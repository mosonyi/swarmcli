// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package contexts

import (
	"swarmcli/docker"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	view.RegisterView(ViewName, factory)
}

func factory(_ docker.Deps, w, h int, _ any) (view.View, tea.Cmd) {
	model := New()
	model.Visible = true
	model.SetSize(w, h)
	model.SetLoading(true)
	return model, tea.Batch(
		func() tea.Msg { return LoadContextsCmd() },
		StartTickerCmd(),
	)
}
