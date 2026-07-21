// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package tasksview

import (
	"github.com/Eldara-Tech/swarmcli/docker"
	"github.com/Eldara-Tech/swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	view.RegisterView(ViewName, factory)
}

func factory(_ docker.Deps, w, h int, payload any) (view.View, tea.Cmd) {
	stackName, _ := payload.(string)
	model := New(w, h, stackName)
	return model, model.OnEnter()
}
