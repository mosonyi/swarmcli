// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"github.com/Eldara-Tech/swarmcli/docker"
	"github.com/Eldara-Tech/swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	view.RegisterView(ViewName, factory)
}

// factory builds the view. It takes no docker.Deps: the release engine is this
// view's data source, and it is reached through the releaseOps seam instead
// (see ops.go for why that cannot live on Deps).
func factory(_ docker.Deps, w, h int, _ any) (view.View, tea.Cmd) {
	m := New(w, h)
	return m, m.Init()
}
