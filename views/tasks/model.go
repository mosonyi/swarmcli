// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package tasksview

import (
	"swarmcli/docker"
	"swarmcli/views/helpbar"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type SortField int

const (
	SortByName SortField = iota
	SortByService
	SortByNode
	SortByState
)

type Model struct {
	viewport      viewport.Model
	visible       bool
	stackName     string
	tasks         []docker.TaskEntry
	width         int
	height        int
	sortField     SortField
	sortAscending bool // true for ascending, false for descending
}

func New(width, height int, stackName string) *Model {
	vp := viewport.New(width, height)
	vp.SetContent("")

	return &Model{
		viewport:      vp,
		visible:       true,
		stackName:     stackName,
		width:         width,
		height:        height,
		sortField:     SortByName,
		sortAscending: true,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Name() string {
	return ViewName
}

func (m *Model) OnEnter() tea.Cmd {
	m.visible = true
	return LoadTasksCmd(m.stackName)
}

func (m *Model) OnExit() tea.Cmd {
	m.visible = false
	return nil
}

func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "↑/↓", Desc: "Scroll"},
		{Key: "shift+n", Desc: "Sort by Name"},
		{Key: "shift+s", Desc: "Sort by Service"},
		{Key: "shift+d", Desc: "Sort by Node"},
		{Key: "shift+t", Desc: "Sort by State"},
		{Key: "Esc", Desc: "Back"},
	}
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.viewport.Width = width
	m.viewport.Height = height
}
