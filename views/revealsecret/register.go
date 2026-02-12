// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package revealsecretview

import (
	"swarmcli/docker"
	"swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	view.RegisterView(ViewName, factory)
}

func factory(_ docker.Deps, w, h int, payload any) (view.View, tea.Cmd) {
	data, _ := payload.(map[string]any)
	secretName, _ := data["secretName"].(string)

	v := New(w, h)
	v.SetSecretName(secretName)
	return v, tea.Batch(v.Init(), LoadSecret(secretName))
}
