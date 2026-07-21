// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package inspectview

import (
	"github.com/Eldara-Tech/swarmcli/docker"
	"github.com/Eldara-Tech/swarmcli/views/view"

	tea "github.com/charmbracelet/bubbletea"
)

func init() {
	view.RegisterView(ViewName, factory)
}

func factory(_ docker.Deps, w, h int, payload any) (view.View, tea.Cmd) {
	data, _ := payload.(map[string]any)
	title, _ := data["title"].(string)
	jsonStr, _ := data["json"].(string)
	raw := ParseFormat(data["format"])

	v := New(w, h, raw)
	return v, LoadInspectItem(title, jsonStr)
}
