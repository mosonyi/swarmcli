// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package view

import (
	"swarmcli/views/helpbar"

	tea "github.com/charmbracelet/bubbletea"
)

type View interface {
	Update(msg tea.Msg) tea.Cmd
	View() string
	Init() tea.Cmd
	Name() string
	ShortHelpItems() []helpbar.HelpEntry

	// Lifecycle hooks:

	OnEnter() tea.Cmd // Called when view becomes active
	OnExit() tea.Cmd  // Called when view is removed/replaced
}
