// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package volumesview

import (
	"github.com/Eldara-Tech/swarmcli/v2/docker"
	"github.com/Eldara-Tech/swarmcli/v2/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	view.RegisterView(ViewName, factory)
}

func factory(deps docker.Deps, w, h int, _ any) (view.View, tea.Cmd) {
	model := New(w, h)
	model.deps = deps
	return model, model.Init()
}
