// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package volumesview

import (
	"fmt"
	"strings"
	"time"

	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/helpbar"
	"swarmcli/views/view"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type SortField int

const (
	SortByName SortField = iota
	SortByStack
	SortByDriver
	SortByCreated
	SortByHost
)

type state int

const (
	stateLoading state = iota
	stateReady
	stateError
)

type Model struct {
	deps          docker.Deps
	volumesList   filterlist.FilterableList[volumeItem]
	width         int
	height        int
	firstResize   bool
	lastSnapshot  uint64
	visible       bool
	sortField     SortField
	sortAscending bool

	resetCursorOnNextLoad bool

	state state
	err   error

	errorDialogActive bool

	spinner int

	toastMessage string
	toastUntil   time.Time

	// partialWarn is a persistent non-fatal banner shown while a cross-node
	// listing is degraded (some nodes unreachable). Cleared on a full load.
	partialWarn string
}

func New(width, height int) *Model {
	vp := viewport.New(width, height)
	vp.SetContent("")

	m := &Model{
		width:         width,
		height:        height,
		firstResize:   true,
		state:         stateLoading,
		visible:       true,
		sortField:     SortByName,
		sortAscending: true,
	}

	list := filterlist.FilterableList[volumeItem]{
		Viewport: vp,
		Match: func(v volumeItem, query string) bool {
			q := strings.ToLower(query)
			return strings.Contains(strings.ToLower(v.Name), q) ||
				strings.Contains(strings.ToLower(v.Stack), q) ||
				strings.Contains(strings.ToLower(v.Driver), q) ||
				strings.Contains(strings.ToLower(v.Mountpoint), q) ||
				strings.Contains(strings.ToLower(v.Host), q)
		},
		Header: &filterlist.HeaderConfig{
			Columns: []filterlist.ColumnDef{
				{Label: "NAME"}, {Label: "STACK"}, {Label: "DRIVER"},
				{Label: "MOUNT POINT"}, {Label: "CREATED"}, {Label: "HOST"},
			},
			ColWidthsFunc: func(w int) []int {
				return m.volumeColWidths(w)
			},
		},
		Footer: &filterlist.FooterConfig{
			ItemLabel: "Volume",
			Override: func(cursor, filteredCount int, mode filterlist.ModeType, query string) string {
				return m.renderVolumesFooter()
			},
		},
	}
	// Non-nil slices so the renderer pads content while loading.
	list.Items = []volumeItem{}
	list.Filtered = []volumeItem{}
	list.SetOuterSize(width, height)

	m.volumesList = list
	m.setRenderItem()
	return m
}

func (m *Model) Name() string { return ViewName }

func (m *Model) Init() tea.Cmd {
	l().Info("VolumesView: Init() called - starting ticker and loading volumes")
	return tea.Batch(tickCmd(), m.spinnerTickCmd(), m.loadVolumesCmd())
}

func (m *Model) spinnerTickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
		return SpinnerTickMsg(t)
	})
}

func tickCmd() tea.Cmd {
	return tea.Tick(PollInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// HasActiveFilter reports whether a filter query is active.
func (m *Model) HasActiveFilter() bool {
	return m.volumesList.Query != ""
}

// CapturesInput reports whether the view is currently capturing all input.
func (m *Model) CapturesInput() bool {
	return m.errorDialogActive
}

// ApplySearchQuery sets the filter query on the volumes list (app-level "/").
func (m *Model) ApplySearchQuery(query string) {
	m.volumesList.Query = query
	m.volumesList.ApplyFilter()
	m.applySorting()
}

// ClearSearchQuery clears the active filter.
func (m *Model) ClearSearchQuery() {
	m.volumesList.Query = ""
	m.volumesList.ApplyFilter()
	m.volumesList.Cursor = 0
	m.volumesList.Viewport.GotoTop()
	m.applySorting()
}

func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "↑/↓", Desc: "Navigate"},
		{Key: "i", Desc: "Inspect"},
		{Key: "c", Desc: view.BEHelpDesc("volume-create", "Create"), Disabled: !view.HasAction("volume-create")},
		{Key: "b", Desc: view.BEHelpDesc("volume-browse", "Browse"), Disabled: !view.HasAction("volume-browse")},
		{Key: "ctrl+d", Desc: view.BEHelpDesc("volume-delete", "Delete"), Disabled: !view.HasAction("volume-delete")},
		{Key: "/", Desc: "Filter"},
		{Key: "?", Desc: "Help"},
		{Key: "esc", Desc: "Back"},
	}
}

func (m *Model) showToast(msg string) {
	if msg == "" {
		return
	}
	m.toastMessage = msg
	d := 2 * time.Second
	if strings.Contains(msg, "\n") {
		d = 5 * time.Second
	}
	m.toastUntil = time.Now().Add(d)
}

func (m *Model) SetVisible(visible bool) {
	m.visible = visible
	l().Info(fmt.Sprintf("VolumesView: SetVisible(%v)", visible))
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.volumesList.Viewport.Width = width
	m.volumesList.Viewport.Height = height
	m.volumesList.SetOuterSize(width, height)
}

func (m *Model) OnEnter() tea.Cmd {
	m.visible = true
	l().Info("VolumesView: OnEnter() - view is now visible")
	m.resetCursorOnNextLoad = true
	m.volumesList.Cursor = 0
	m.volumesList.Viewport.YOffset = 0
	return m.loadVolumesCmd()
}

func (m *Model) OnExit() tea.Cmd {
	m.visible = false
	l().Info("VolumesView: OnExit() - view is no longer visible")
	return nil
}

func (m *Model) HasErrors() bool { return false }
