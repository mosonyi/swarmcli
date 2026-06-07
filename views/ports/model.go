// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package portsview

import (
	"swarmcli/docker"
	"swarmcli/views/helpbar"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	deps     docker.Deps
	viewport viewport.Model
	width    int
	height   int
	ready    bool
}

func New(width, height int) *Model {
	vp := viewport.New(width, height)
	vp.SetContent("")
	return &Model{
		viewport: vp,
		width:    width,
		height:   height,
	}
}

func (m *Model) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(PollInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m *Model) Name() string {
	return ViewName
}

func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "esc", Desc: "Back"},
	}
}

func (m *Model) OnEnter() tea.Cmd {
	return nil
}

func (m *Model) OnExit() tea.Cmd {
	return nil
}

func (m *Model) HasErrors() bool {
	return false
}
