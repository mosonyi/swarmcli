// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package loadingview

import (
	"github.com/Eldara-Tech/swarmcli/v2/docker"
	"github.com/Eldara-Tech/swarmcli/v2/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	view.RegisterView(ViewName, factory)
}

func factory(_ docker.Deps, w, h int, payload any) (view.View, tea.Cmd) {
	return New(w, h, true, payload), nil
}
