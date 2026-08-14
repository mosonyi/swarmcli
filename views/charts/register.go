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
//
// An optional {"release": name} payload arrives from a cross-link — the stacks
// view's jump to the release that owns a stack — and selects that release once
// the first read lands.
func factory(_ docker.Deps, w, h int, payload any) (view.View, tea.Cmd) {
	m := New(w, h)
	if data, ok := payload.(map[string]any); ok {
		if name, ok := data["release"].(string); ok && name != "" {
			m.pendingSelect = name
		}
	}
	return m, m.Init()
}
