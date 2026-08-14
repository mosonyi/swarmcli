// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package networksview

import (
	"fmt"
	"github.com/Eldara-Tech/swarmcli/docker"
	filterlist "github.com/Eldara-Tech/swarmcli/ui/components/filterable/list"
	"github.com/Eldara-Tech/swarmcli/views/confirmdialog"
	"github.com/Eldara-Tech/swarmcli/views/helpbar"
	loading "github.com/Eldara-Tech/swarmcli/views/loading"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type SortField int

const (
	SortByName SortField = iota
	SortByID
	SortByDriver
	SortByScope
	SortByUsed
	SortByCreated
)

type Model struct {
	deps                  docker.Deps
	networksList          filterlist.FilterableList[networkItem]
	width                 int
	height                int
	firstResize           bool   // tracks if we've received the first window size
	lastSnapshot          uint64 // hash of last snapshot for change detection
	visible               bool   // tracks if view is currently active
	resetCursorOnNextLoad bool   // one-shot: force cursor to top on next NetworksLoadedMsg
	sortField             SortField
	sortAscending         bool // true for ascending, false for descending

	state state
	err   error

	pendingAction     string
	confirmDialog     *confirmdialog.Model
	errorDialogActive bool
	loadingView       *loading.Model
	networks          []networkItem
	networkToDelete   *networkItem

	// Used By view
	usedByViewActive  bool
	usedByList        filterlist.FilterableList[usedByItem]
	usedByNetworkName string

	// Cached column widths for header alignment
	colNameWidth   int
	colDriverWidth int
	colScopeWidth  int

	// Spinner for loading indicator
	spinner int

	// Small transient status message shown in footer.
	toastMessage string
	toastUntil   time.Time

	// Create network dialog
	createDialogActive bool
	createDialogStep   string // "basic" or "review"
	createDialogError  string
	createInputFocus   int // 0=name, 1=driver, 2=ipv4 subnet, 3=ipv4 gateway, 4=enable ipv6, 5=ipv6 subnet, 6=ipv6 gateway, 7=isolated(internal), 8=manual attachment(attachable)
	createNameInput    textinput.Model
	createIPv4Subnet   textinput.Model
	createIPv4Gateway  textinput.Model
	createEnableIPv6   bool
	createIPv6Subnet   textinput.Model
	createIPv6Gateway  textinput.Model
	createDriverIndex  int
	createAttachable   bool
	createInternal     bool
}

type state int

const (
	stateLoading state = iota
	stateReady
	stateError
)

func New(width, height int) *Model {
	vp := viewport.New(width, height)
	vp.SetContent("")

	nameInput := textinput.New()
	nameInput.Placeholder = "my-network"
	nameInput.Prompt = "Name: "
	nameInput.CharLimit = 128
	nameInput.Width = 50

	ipv4Subnet := textinput.New()
	ipv4Subnet.Placeholder = "10.0.0.0/24"
	ipv4Subnet.Prompt = "IPv4 Subnet: "
	ipv4Subnet.CharLimit = 64
	ipv4Subnet.Width = 50

	ipv4Gateway := textinput.New()
	ipv4Gateway.Placeholder = "10.0.0.1"
	ipv4Gateway.Prompt = "IPv4 Gateway: "
	ipv4Gateway.CharLimit = 64
	ipv4Gateway.Width = 50

	ipv6Subnet := textinput.New()
	ipv6Subnet.Placeholder = "fd00::/64"
	ipv6Subnet.Prompt = "IPv6 Subnet: "
	ipv6Subnet.CharLimit = 64
	ipv6Subnet.Width = 50

	ipv6Gateway := textinput.New()
	ipv6Gateway.Placeholder = "fd00::1"
	ipv6Gateway.Prompt = "IPv6 Gateway: "
	ipv6Gateway.CharLimit = 64
	ipv6Gateway.Width = 50

	m := &Model{
		width:             width,
		height:            height,
		firstResize:       true,
		state:             stateLoading,
		visible:           true,
		confirmDialog:     confirmdialog.New(0, 0),
		loadingView:       loading.New(width, height, false, "Loading Docker networks..."),
		sortField:         SortByName,
		sortAscending:     true,
		createNameInput:   nameInput,
		createIPv4Subnet:  ipv4Subnet,
		createIPv4Gateway: ipv4Gateway,
		createEnableIPv6:  false,
		createIPv6Subnet:  ipv6Subnet,
		createIPv6Gateway: ipv6Gateway,
	}

	list := filterlist.FilterableList[networkItem]{
		Viewport: vp,
		Match: func(n networkItem, query string) bool {
			q := strings.ToLower(query)
			return strings.Contains(strings.ToLower(n.Name), q) ||
				strings.Contains(strings.ToLower(n.ID), q) ||
				strings.Contains(strings.ToLower(n.Driver), q) ||
				strings.Contains(strings.ToLower(n.Scope), q)
		},
		Header: &filterlist.HeaderConfig{
			Columns: []filterlist.ColumnDef{
				{Label: "NAME"}, {Label: "DRIVER"}, {Label: "SCOPE"}, {Label: "USED"}, {Label: "ID"},
			},
			ColWidthsFunc: func(w int) []int {
				nameW, driverW, scopeW, usedW, idW := m.networkColWidths(w)
				return []int{nameW, driverW, scopeW, usedW, idW}
			},
			SortIndicator: func() (int, bool) {
				colMap := map[SortField]int{
					SortByName: 0, SortByDriver: 1, SortByScope: 2, SortByUsed: 3, SortByID: 4,
				}
				col, ok := colMap[m.sortField]
				if !ok {
					return -1, true
				}
				return col, m.sortAscending
			},
		},
		Footer: &filterlist.FooterConfig{
			ItemLabel: "Network",
			Override: func(cursor, filteredCount int, mode filterlist.ModeType, query string) string {
				return m.renderNetworksFooter()
			},
		},
	}
	// Important: make Items a non-nil slice so the FilterableList renderer pads
	// content properly while loading (avoids truncated overlays / missing rows).
	list.Items = []networkItem{}
	list.Filtered = []networkItem{}
	list.SetOuterSize(width, height)

	m.networksList = list
	return m
}

func (m *Model) Name() string { return ViewName }

// HasActiveFilter reports whether a filter query is active.
func (m *Model) HasActiveFilter() bool {
	return m.networksList.Query != ""
}

// CapturesInput reports whether the view is currently capturing all keyboard input.
func (m *Model) CapturesInput() bool {
	if m.confirmDialog != nil && m.confirmDialog.Visible {
		return true
	}
	if m.createDialogActive {
		return true
	}
	if m.errorDialogActive {
		return true
	}
	return false
}

// IsSearching reports whether the networks view is in a sub-view that should
// capture keys (usedBy, createDialog).
func (m *Model) IsSearching() bool {
	if m.usedByViewActive {
		return true
	}
	if m.createDialogActive {
		return true
	}
	return false
}

// ApplySearchQuery sets the filter query on the primary networks list.
func (m *Model) ApplySearchQuery(query string) {
	m.networksList.Query = query
	m.networksList.ApplyFilter()
}

// ClearSearchQuery clears the filter on the primary networks list.
func (m *Model) ClearSearchQuery() {
	m.networksList.Query = ""
	m.networksList.ApplyFilter()
	m.networksList.Cursor = 0
	m.networksList.Viewport.GotoTop()
}

func (m *Model) Init() tea.Cmd {
	l().Info("NetworksView: Init() called - starting ticker and loading networks")
	return tea.Batch(m.spinnerTickCmd(), m.loadNetworksCmd())
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

func (m *Model) LoadNetworks() tea.Cmd {
	return m.loadNetworksCmd()
}

func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	if m.usedByViewActive {
		return []helpbar.HelpEntry{
			{Key: "↑/↓", Desc: "Navigate"},
			{Key: "Enter", Desc: "Go to Service"},
			{Key: "/", Desc: "Filter"},
			{Key: "Esc", Desc: "Back"},
		}
	}

	return []helpbar.HelpEntry{
		{Key: "↑/↓", Desc: "Navigate"},
		{Key: "c", Desc: "Create"},
		{Key: "i", Desc: "Inspect"},
		{Key: "u", Desc: "Used By"},
		{Key: "ctrl+d", Desc: "Delete"},
		{Key: "ctrl+u", Desc: "Prune Unused"},
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
	// Give multi-line toasts a bit more time so users can read them.
	d := 2 * time.Second
	if strings.Contains(msg, "\n") {
		d = 5 * time.Second
	}
	m.toastUntil = time.Now().Add(d)
}

func (m *Model) SetVisible(visible bool) {
	m.visible = visible
	l().Info(fmt.Sprintf("NetworksView: SetVisible(%v)", visible))
}

func (m *Model) SetSize(width, height int) {
	m.width = width
	m.height = height

	if m.confirmDialog != nil {
		m.confirmDialog.Width = width
		m.confirmDialog.Height = height
	}
	if m.loadingView != nil {
		m.loadingView.SetSize(width, height)
	}

	// Resize list viewports. Height is already adjusted by the app;
	// do not subtract header/footer/help again.
	m.networksList.Viewport.Width = width
	m.networksList.Viewport.Height = height
	m.networksList.SetOuterSize(width, height)

	if m.usedByViewActive {
		m.usedByList.Viewport.Width = width
		m.usedByList.Viewport.Height = height
		m.usedByList.SetOuterSize(width, height)
	}
}

func (m *Model) OnEnter() tea.Cmd {
	m.visible = true
	l().Info("NetworksView: OnEnter() - view is now visible")
	// When entering the view, prefer a predictable UX: start at the top.
	// We keep cursor-restore behavior for background refreshes, but suppress it
	// for the first load after entering.
	m.resetCursorOnNextLoad = true
	m.networksList.Cursor = 0
	m.networksList.Viewport.YOffset = 0
	// The tick is armed here, not in Init or the factory: OnEnter is the only
	// hook that runs both on first entry and on every return from a drill-down,
	// and a chain cannot survive a navigation (see the TickMsg handler).
	return tea.Batch(m.LoadNetworks(), tickCmd())
}

func (m *Model) OnExit() tea.Cmd {
	m.visible = false
	l().Info("NetworksView: OnExit() - view is no longer visible")
	return nil
}

func (m *Model) HasErrors() bool {
	return false
}
