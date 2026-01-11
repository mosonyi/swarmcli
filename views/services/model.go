package servicesview

import (
	"strings"
	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/confirmdialog"
	"swarmcli/views/helpbar"
	"swarmcli/views/scaledialog"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

const ViewName = "services"

type FilterType int

const (
	NodeFilter FilterType = iota
	StackFilter
	AllFilter
)

type Model struct {
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

	confirmDialog *confirmdialog.Model
	scaleDialog   *scaledialog.Model

	// Track what action is pending confirmation
	pendingAction string // "restart", "remove", "rollback", or "empty-stack"

	// Track which services have their tasks expanded
	expandedServices map[string]bool               // service ID -> expanded
	serviceTasks     map[string][]docker.TaskEntry // cached tasks per service

	// Track task navigation: -1 means service row is selected, >= 0 means task at that index
	selectedTaskIndex int
}

func New(width, height int) *Model {
	vp := viewport.New(width, height)

	list := filterlist.FilterableList[docker.ServiceEntry]{
		Viewport: vp,
		Match: func(s docker.ServiceEntry, query string) bool {
			return strings.Contains(strings.ToLower(s.ServiceName), strings.ToLower(query))
		},
	}

	return &Model{
		List:              list,
		Visible:           false,
		firstResize:       true,
		width:             width,
		height:            height,
		confirmDialog:     confirmdialog.New(width, height),
		scaleDialog:       scaledialog.New(width, height),
		expandedServices:  make(map[string]bool),
		serviceTasks:      make(map[string][]docker.TaskEntry),
		selectedTaskIndex: -1,
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
	return m.List.Mode == filterlist.ModeSearching
}

// HasActiveDialog reports whether a dialog is currently visible.
func (m *Model) HasActiveDialog() bool {
	return m.confirmDialog.Visible || m.scaleDialog.Visible
}
