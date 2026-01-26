// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package networksview

import (
	"fmt"
	"strings"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/confirmdialog"
	"swarmcli/views/helpbar"
	loading "swarmcli/views/loading"
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

	// Inspect view
	inspectViewActive bool
	inspectViewport   viewport.Model
	inspectContent    string
	inspectSearchMode bool
	inspectSearchTerm string

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

	list := filterlist.FilterableList[networkItem]{
		Viewport: vp,
		Match: func(n networkItem, query string) bool {
			q := strings.ToLower(query)
			return strings.Contains(strings.ToLower(n.Name), q) ||
				strings.Contains(strings.ToLower(n.ID), q) ||
				strings.Contains(strings.ToLower(n.Driver), q) ||
				strings.Contains(strings.ToLower(n.Scope), q)
		},
	}
	// Important: make Items a non-nil slice so the FilterableList renderer pads
	// content properly while loading (avoids truncated overlays / missing rows).
	list.Items = []networkItem{}
	list.Filtered = []networkItem{}

	inspectVp := viewport.New(width, height)
	inspectVp.SetContent("")

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

	return &Model{
		networksList:      list,
		width:             width,
		height:            height,
		firstResize:       true,
		state:             stateLoading,
		visible:           true,
		confirmDialog:     confirmdialog.New(0, 0),
		loadingView:       loading.New(width, height, false, "Loading Docker networks..."),
		inspectViewport:   inspectVp,
		sortField:         SortByName,
		sortAscending:     true,
		createNameInput:   nameInput,
		createIPv4Subnet:  ipv4Subnet,
		createIPv4Gateway: ipv4Gateway,
		createEnableIPv6:  false,
		createIPv6Subnet:  ipv6Subnet,
		createIPv6Gateway: ipv6Gateway,
	}
}

func (m *Model) Name() string { return ViewName }

// HasActiveFilter reports whether a filter query is active.
func (m *Model) HasActiveFilter() bool {
	return m.networksList.Query != ""
}

// HasActiveDialog reports whether Networks currently has a modal dialog open.
// The app uses this to route key handling to the view instead of performing
// global navigation on ESC.
func (m *Model) HasActiveDialog() bool {
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

// IsSearching reports whether the networks or UsedBy list is in search mode.
func (m *Model) IsSearching() bool {
	// Important: app-level key handling uses IsSearching() to decide whether ESC/Q
	// should be handled by the view or should pop the global view stack.
	// Networks has internal sub-views (inspect/used-by) that must consume ESC/Q.
	if m.inspectViewActive {
		return true
	}
	if m.usedByViewActive {
		return true
	}
	if m.createDialogActive {
		return true
	}
	return m.networksList.Mode == filterlist.ModeSearching
}

func (m *Model) Init() tea.Cmd {
	l().Info("NetworksView: Init() called - starting ticker and loading networks")
	return tea.Batch(tickCmd(), m.spinnerTickCmd(), LoadNetworks())
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

func LoadNetworks() tea.Cmd {
	return loadNetworksCmd()
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

	if m.inspectViewActive {
		if m.inspectSearchMode {
			return []helpbar.HelpEntry{
				{Key: "Type", Desc: "Search"},
				{Key: "Enter", Desc: "Apply"},
				{Key: "Esc", Desc: "Cancel"},
			}
		}
		return []helpbar.HelpEntry{
			{Key: "/", Desc: "Search"},
			{Key: "↑/↓/PgUp/PgDn", Desc: "Scroll"},
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
		{Key: "esc/q", Desc: "Back"},
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

	// Resize inspect viewport (content area inside the frame).
	contentH := height - 4 // top+bottom borders + header + footer
	if contentH < 1 {
		contentH = 1
	}
	m.inspectViewport.Width = width
	m.inspectViewport.Height = contentH

	// Resize list viewports. Height is already adjusted by the app;
	// do not subtract header/footer/help again.
	m.networksList.Viewport.Width = width
	m.networksList.Viewport.Height = height

	if m.usedByViewActive {
		m.usedByList.Viewport.Width = width
		m.usedByList.Viewport.Height = height
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
	return LoadNetworks()
}

func (m *Model) OnExit() tea.Cmd {
	m.visible = false
	l().Info("NetworksView: OnExit() - view is no longer visible")
	return nil
}
