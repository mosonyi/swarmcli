// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package stacksview

import (
	"swarmcli/docker"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	view.RegisterView(ViewName, factory)
}

func factory(deps docker.Deps, w, h int, payload any) (view.View, tea.Cmd) {
	var nodeID string
	if payload != nil {
		nodeID, _ = payload.(string)
	}
	model := New(w, h)
	model.deps = deps
	model.Visible = true
	return model, tea.Batch(model.Init(), model.LoadStacksCmd(nodeID))
}
