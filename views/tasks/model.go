// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package tasksview

import (
	"github.com/Eldara-Tech/swarmcli/v2/docker"
	"github.com/Eldara-Tech/swarmcli/v2/views/helpbar"

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
	sortAscending bool   // true for ascending, false for descending
	lastSnapshot  uint64 // hash of last snapshot for change detection
	pollGen       uint64 // generation of the live poll chain; see OnEnter
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
func (m *Model) HasErrors() bool {
	return false
}
func (m *Model) Name() string {
	return ViewName
}

func (m *Model) OnEnter() tea.Cmd {
	m.visible = true
	// The tick is armed here, not in Init or the factory: OnEnter is the only
	// hook that runs both on first entry and on every return from a drill-down,
	// and a chain does not survive a navigation — its tick is delivered to
	// whichever view is current by then, and dropped.
	//
	// Each entry gets its own generation. "Does not survive" holds only once
	// the leftover tick has fired: one armed just before a drill-down can
	// still be in flight when the operator returns, and would find this view
	// current again and re-arm, leaving two chains for the rest of the view's
	// life. The generation makes it recognisable as a leftover.
	m.pollGen++
	return tea.Batch(LoadTasksCmd(m.stackName), tickCmd(m.pollGen))
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
