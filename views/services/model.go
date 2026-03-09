// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package servicesview

import (
	"strings"
	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/confirmdialog"
	"swarmcli/views/helpbar"
	"swarmcli/views/scaledialog"
	"swarmcli/views/view"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

const ViewName = "services"

type FilterType int

const (
	NodeFilter FilterType = iota
	StackFilter
	NoStackFilter
	AllFilter
)

type SortField int

const (
	SortByName SortField = iota
	SortByStatus
	SortByImage
	SortByPorts
	SortByCreated
	SortByUpdated
	SortByError
)

type Model struct {
	deps         docker.Deps
	List         filterlist.FilterableList[docker.ServiceEntry]
	Visible      bool
	title        string
	ready        bool
	firstResize  bool // tracks if we've received the first window size
	width        int
	height       int
	lastSnapshot uint64 // hash of last snapshot for change detection

	// Column widths cached after computation
	colServiceWidth int
	colStackWidth   int

	// Filter
	filterType FilterType
	nodeID     string
	stackName  string

	// Optional one-shot selection after navigation.
	pendingSelectServiceName string

	confirmDialog *confirmdialog.Model
	scaleDialog   *scaledialog.Model

	// Track what action is pending confirmation
	pendingAction string // "restart", "remove", "rollback", or "empty-stack"

	// Track which services have their tasks expanded
	expandedServices map[string]bool               // service ID -> expanded
	serviceTasks     map[string][]docker.TaskEntry // cached tasks per service
	// serviceHasError marks if a service has a running task with an error
	serviceHasError map[string]bool
	// serviceErrorText stores a representative error text per service
	serviceErrorText map[string]string
	// columnScrollOffset controls horizontal scroll for all columns
	columnScrollOffset int

	// Track task navigation: -1 means service row is selected, >= 0 means task at that index
	selectedTaskIndex int

	// Sorting
	sortField     SortField
	sortAscending bool // true for ascending, false for descending
}

func (m *Model) SetPendingSelectServiceName(serviceName string) {
	m.pendingSelectServiceName = serviceName
}

func New(width, height int) *Model {
	vp := viewport.New(width, height)

	m := &Model{
		Visible:           false,
		firstResize:       true,
		width:             width,
		height:            height,
		confirmDialog:     confirmdialog.New(width, height),
		scaleDialog:       scaledialog.New(width, height),
		expandedServices:  make(map[string]bool),
		serviceTasks:      make(map[string][]docker.TaskEntry),
		serviceHasError:   make(map[string]bool),
		serviceErrorText:  make(map[string]string),
		selectedTaskIndex: -1,
		sortField:         SortByName,
		sortAscending:     true,
	}

	list := filterlist.FilterableList[docker.ServiceEntry]{
		Viewport: vp,
		Match: func(s docker.ServiceEntry, query string) bool {
			return strings.Contains(strings.ToLower(s.ServiceName), strings.ToLower(query))
		},
		Header: &filterlist.HeaderConfig{
			Columns: []filterlist.ColumnDef{
				{Label: "SERVICE"}, {Label: "STACK"}, {Label: "REPLICAS"},
				{Label: "STATUS"}, {Label: "MODE"}, {Label: "IMAGE"},
				{Label: "PORTS"}, {Label: "CREATED"}, {Label: "UPDATED"}, {Label: "ERROR"},
			},
			ColWidthsFunc: m.computeColWidths,
			SortIndicator: func() (int, bool) {
				colMap := map[SortField]int{
					SortByName: 0, SortByStatus: 3, SortByImage: 5,
					SortByPorts: 6, SortByCreated: 7, SortByUpdated: 8, SortByError: 9,
				}
				col, ok := colMap[m.sortField]
				if !ok {
					return -1, true
				}
				return col, m.sortAscending
			},
		},
		Footer: &filterlist.FooterConfig{ItemLabel: "Node"},
	}
	list.SetOuterSize(width, height)

	m.List = list
	return m
}

func (m *Model) Init() tea.Cmd {
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(PollInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m *Model) Name() string { return ViewName }

func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "i", Desc: "Inspect"},
		{Key: "↑/↓", Desc: "Navigate"},
		{Key: "p", Desc: "Show/hide tasks"},
		{Key: "s", Desc: "Scale service"},
		{Key: "r", Desc: "Restart service"},
		{Key: "ctrl+r", Desc: "Rollback service"},
		{Key: "ctrl+d", Desc: "Remove service"},
		{Key: "l", Desc: "View logs"},
		{Key: "x", Desc: view.BEHelpDesc("shell", "Shell"), Disabled: !view.HasAction("shell")},
		{Key: "w", Desc: view.BEHelpDesc("port-forward", "Port Forward"), Disabled: !view.HasAction("port-forward")},
		{Key: "/", Desc: "Filter"},
		{Key: "?", Desc: "Help"},
		{Key: "q", Desc: "Close"},
	}
}

func (m *Model) OnEnter() tea.Cmd {
	return nil
}

func (m *Model) OnExit() tea.Cmd {
	return nil
}

// HasActiveFilter reports whether a filter query is active.
func (m *Model) HasActiveFilter() bool {
	return m.List.Query != ""
}

// IsSearching reports whether the list is currently in search mode.
func (m *Model) IsSearching() bool {
	return false
}

// ApplySearchQuery sets the filter query and applies it.
func (m *Model) ApplySearchQuery(query string) {
	m.List.Query = query
	m.List.ApplyFilter()
}

// ClearSearchQuery clears the filter query and resets the view.
func (m *Model) ClearSearchQuery() {
	m.List.Query = ""
	m.List.ApplyFilter()
	m.List.Cursor = 0
	m.List.Viewport.GotoTop()
}

// HasActiveDialog reports whether a dialog is currently visible.
func (m *Model) HasActiveDialog() bool {
	return m.confirmDialog.Visible || m.scaleDialog.Visible
}

// HasErrors returns true if any service in the current filtered list has errors
func (m *Model) HasErrors() bool {
	// Only check errors for services that are actually in the current view
	for _, svc := range m.List.Filtered {
		if m.serviceHasError[svc.ServiceID] {
			return true
		}
	}
	return false
}
