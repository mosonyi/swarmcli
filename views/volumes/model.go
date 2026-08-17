// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package volumesview

import (
	"fmt"
	"strings"
	"time"

	"github.com/Eldara-Tech/swarmcli/docker"
	filterlist "github.com/Eldara-Tech/swarmcli/ui/components/filterable/list"
	"github.com/Eldara-Tech/swarmcli/views/helpbar"
	"github.com/Eldara-Tech/swarmcli/views/view"

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
	pollGen       uint64 // generation of the live poll chain; see OnEnter
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

	cols := m.buildColumns()
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
		Columns: cols,
		Header: &filterlist.HeaderConfig{
			Columns: filterlist.ColumnDefs(cols),
			SortIndicator: func() (int, bool) {
				return m.sortColumnIndex(), m.sortAscending
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
	return tea.Batch(m.spinnerTickCmd(), m.loadVolumesCmd())
}

func (m *Model) spinnerTickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
		return SpinnerTickMsg(t)
	})
}

func tickCmd(gen uint64) tea.Cmd {
	return tea.Tick(PollInterval, func(time.Time) tea.Msg {
		return TickMsg{Gen: gen}
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
		{Key: "p", Desc: view.BEHelpDesc("volume-prune", "Prune"), Disabled: !view.HasAction("volume-prune")},
		{Key: "/", Desc: "Filter"},
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
	return tea.Batch(m.loadVolumesCmd(), tickCmd(m.pollGen))
}

func (m *Model) OnExit() tea.Cmd {
	m.visible = false
	l().Info("VolumesView: OnExit() - view is no longer visible")
	return nil
}

func (m *Model) HasErrors() bool { return false }
