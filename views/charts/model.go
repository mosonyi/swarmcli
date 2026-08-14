// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package chartsview

import (
	"strings"
	"time"

	filterlist "github.com/Eldara-Tech/swarmcli/ui/components/filterable/list"
	"github.com/Eldara-Tech/swarmcli/views/confirmdialog"
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

	// pendingSelect is a release a cross-link asked for, applied once the first
	// read lands — the view is empty when the factory runs, so there is nothing
	// to select yet.
	pendingSelect string

	// expanded tracks which releases show their revisions and services inline,
	// keyed by release name so a background reload cannot collapse them.
	expanded map[string]bool
	// childIndex is the selected child within the expanded release under the
	// cursor, or noChild when the release row itself is selected.
	childIndex int

	// haveIndexes reports whether any cached repository index backed the last
	// read. Without one the LATEST column is empty for every release, which must
	// not be mistaken for "everything is up to date".
	haveIndexes bool

	state state
	err   error

	errorDialogActive bool

	// confirmDialog is only ever used in InfoMode: this view has nothing to
	// confirm, only blocked actions to explain.
	confirmDialog *confirmdialog.Model

	// tickScheduled guards the poll ticker. Exactly one tick may be
	// outstanding: every place that re-arms is a chain, and two chains do not
	// merely double the poll rate, they each double again on every beat.
	tickScheduled bool

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
		expanded:      map[string]bool{},
		childIndex:    noChild,
		confirmDialog: confirmdialog.New(width, height),
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
	return tea.Batch(m.armTick(), spinnerTickCmd(), m.loadReleasesCmd())
}

// armTick schedules the next poll, unless one is already scheduled.
//
// The guard is the whole point. A tick that both starts a poll and re-arms,
// while the poll's result re-arms too, produces two successors per beat — and
// each of those does the same, so an idle view climbs to thousands of
// concurrent reads within a minute. Re-arming only after the poll's result has
// the further benefit that two polls can never overlap.
func (m *Model) armTick() tea.Cmd {
	if m.tickScheduled {
		return nil
	}
	m.tickScheduled = true
	return tickCmd()
}

// HasActiveFilter reports whether a filter query is active.
func (m *Model) HasActiveFilter() bool { return m.list.Query != "" }

// CapturesInput reports whether the view is currently capturing all input.
func (m *Model) CapturesInput() bool { return m.errorDialogActive || m.confirmDialog.Visible }

// ApplySearchQuery sets the filter query on the release list (app-level "/").
func (m *Model) ApplySearchQuery(query string) {
	m.list.Query = query
	m.list.ApplyFilter()
	m.childIndex = noChild
	m.applySorting()
}

// ClearSearchQuery clears the active filter.
func (m *Model) ClearSearchQuery() {
	m.list.Query = ""
	m.list.ApplyFilter()
	m.list.Cursor = 0
	m.childIndex = noChild
	m.list.Viewport.GotoTop()
	m.applySorting()
}

func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "↑/↓", Desc: "Navigate"},
		{Key: "enter", Desc: "Expand"},
		{Key: "i", Desc: "Manifest"},
		{Key: "v", Desc: "Values"},
		{Key: "d", Desc: "Diff"},
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
	m.confirmDialog.Width = width
	m.confirmDialog.Height = height
}

func (m *Model) OnEnter() tea.Cmd {
	m.visible = true
	// A cross-link's requested release outranks the usual jump to the top:
	// the app calls OnEnter after the factory, so resetting here would undo
	// the selection the payload asked for.
	if m.pendingSelect == "" {
		m.resetCursorOnNextLoad = true
		m.list.Cursor = 0
		m.list.Viewport.YOffset = 0
		// The child selection belongs to whichever release the cursor was on.
		// Leaving it set while the cursor moves to the top highlights nothing
		// on screen and swallows the operator's next esc.
		m.childIndex = noChild
	}
	return tea.Batch(m.loadReleasesCmd(), m.armTick())
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

// isExpanded reports whether the release under the cursor shows its children.
func (m *Model) isExpanded() bool {
	sel, ok := m.selected()
	return ok && m.expanded[sel.Name]
}

// selectedChild returns the child row under the cursor, if one is selected.
func (m *Model) selectedChild() (childRow, bool) {
	if m.childIndex == noChild {
		return childRow{}, false
	}
	sel, ok := m.selected()
	if !ok || !m.expanded[sel.Name] {
		return childRow{}, false
	}
	rows := sel.children()
	if m.childIndex < 0 || m.childIndex >= len(rows) {
		return childRow{}, false
	}
	return rows[m.childIndex], true
}

// IsRowExpanded tells the app that esc has somewhere to go inside this view:
// back to the release row from a child, then collapsing the release. Without
// it the app's esc chain would pop straight out of the view.
//
// Deliberately not routed through CapturesInput, which the ":" handler also
// consults and which would disable the command bar while a row is expanded.
func (m *Model) IsRowExpanded() bool {
	return m.childIndex != noChild || m.isExpanded()
}
