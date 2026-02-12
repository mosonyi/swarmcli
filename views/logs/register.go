// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package logsview

import (
	"swarmcli/docker"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	view.RegisterView(ViewName, factory)
}

func factory(_ docker.Deps, w, h int, payload any) (view.View, tea.Cmd) {
	service := payload.(docker.ServiceEntry)
	v := New(w, h, 10000, service)
	return v, StartStreamingCmd(v.StreamCtx, service, 200, v.MaxLines)
}
