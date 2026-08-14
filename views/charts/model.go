// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"strings"
	"time"

	filterlist "github.com/Eldara-Tech/swarmcli/ui/components/filterable/list"
	"github.com/Eldara-Tech/swarmcli/views/helpbar"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type SortField int

const (
	SortByName SortField = iota
	SortByRevision
	SortByStatus
	SortByHealth
	SortByChart
	SortByUpdated
)

type state int

const (
	stateLoading state = iota
	stateReady
	stateError
)

type Model struct {
	ops           releaseOps
	list          filterlist.FilterableList[releaseItem]
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
}

func New(width, height int) *Model {
	vp := viewport.New(width, height)
	vp.SetContent("")

	m := &Model{
		ops:           newEngineOps(),
		width:         width,
		height:        height,
		firstResize:   true,
		state:         stateLoading,
		visible:       true,
		sortField:     SortByName,
		sortAscending: true,
	}

	cols := m.buildColumns()
	list := filterlist.FilterableList[releaseItem]{
		Viewport: vp,
		Match: func(r releaseItem, query string) bool {
			q := strings.ToLower(query)
			return strings.Contains(strings.ToLower(r.Name), q) ||
				strings.Contains(strings.ToLower(r.chartRef()), q)
		},
		Columns: cols,
		Header: &filterlist.HeaderConfig{
			Columns: filterlist.ColumnDefs(cols),
			SortIndicator: func() (int, bool) {
				return m.sortColumnIndex(), m.sortAscending
			},
		},
		Footer: &filterlist.FooterConfig{
			ItemLabel: "Release",
			Override: func(cursor, filteredCount int, mode filterlist.ModeType, query string) string {
				return m.renderFooter()
			},
		},
	}
	// Non-nil slices so the renderer pads content while loading.
	list.Items = []releaseItem{}
	list.Filtered = []releaseItem{}
	list.SetOuterSize(width, height)

	m.list = list
	m.setRenderItem()
	return m
}

func (m *Model) Name() string { return ViewName }

func (m *Model) Init() tea.Cmd {
	return tea.Batch(tickCmd(), spinnerTickCmd(), m.loadReleasesCmd())
}

// HasActiveFilter reports whether a filter query is active.
func (m *Model) HasActiveFilter() bool { return m.list.Query != "" }

// CapturesInput reports whether the view is currently capturing all input.
func (m *Model) CapturesInput() bool { return m.errorDialogActive }

// ApplySearchQuery sets the filter query on the release list (app-level "/").
func (m *Model) ApplySearchQuery(query string) {
	m.list.Query = query
	m.list.ApplyFilter()
	m.applySorting()
}

// ClearSearchQuery clears the active filter.
func (m *Model) ClearSearchQuery() {
	m.list.Query = ""
	m.list.ApplyFilter()
	m.list.Cursor = 0
	m.list.Viewport.GotoTop()
	m.applySorting()
}

func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "↑/↓", Desc: "Navigate"},
		{Key: "i", Desc: "Manifest"},
		{Key: "v", Desc: "Values"},
		{Key: "s", Desc: "Services"},
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

func (m *Model) SetVisible(visible bool) { m.visible = visible }

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height
	m.list.Viewport.Width = width
	m.list.Viewport.Height = height
	m.list.SetOuterSize(width, height)
}

func (m *Model) OnEnter() tea.Cmd {
	m.visible = true
	m.resetCursorOnNextLoad = true
	m.list.Cursor = 0
	m.list.Viewport.YOffset = 0
	return m.loadReleasesCmd()
}

func (m *Model) OnExit() tea.Cmd {
	m.visible = false
	return nil
}

func (m *Model) HasErrors() bool { return false }

// selected returns the release under the cursor, or false if the list is empty.
func (m *Model) selected() (releaseItem, bool) {
	if m.list.Cursor < 0 || m.list.Cursor >= len(m.list.Filtered) {
		return releaseItem{}, false
	}
	return m.list.Filtered[m.list.Cursor], true
}
