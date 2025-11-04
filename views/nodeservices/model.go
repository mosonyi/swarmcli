package nodeservicesview

import (
	"fmt"
	"swarmcli/views/confirmdialog"
	"swarmcli/views/helpbar"
	loadingview "swarmcli/views/loading"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type FilterType int

const (
	NodeFilter FilterType = iota
	StackFilter
)

type Model struct {
	viewport viewport.Model
	Visible  bool

	entries []ServiceEntry
	cursor  int
	title   string
	ready   bool

	serviceColWidth int
	stackColWidth   int
	replicaColWidth int

	// Filter
	filterType FilterType
	nodeID     string
	hostname   string
	stackName  string

	confirmDialog confirmdialog.Model
	loading       loadingview.Model
}

type ServiceEntry struct {
	StackName      string
	ServiceName    string
	ServiceID      string
	ReplicasOnNode int
	ReplicasTotal  int
}

// Create new instance
func New(width, height int) Model {
	vp := viewport.New(width, height)
	ld := loadingview.New(width, height, false, "Please wait...")
	return Model{
		viewport:      vp,
		Visible:       false,
		confirmDialog: confirmdialog.New(width, height),
		loading:       ld,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Name() string { return ViewName }

func (m Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "i", Desc: "inspect"},
		{Key: "k/up", Desc: "up"},
		{Key: "r", Desc: "restart service"},
		{Key: "j/down", Desc: "down"},
		{Key: "q", Desc: "close"},
	}
}

func (m *Model) SetContent(msg Msg) {
	m.title = msg.Title
	m.entries = msg.Entries
	m.cursor = 0
	m.filterType = msg.FilterType
	m.nodeID = msg.NodeID
	m.hostname = msg.Hostname
	m.stackName = msg.StackName

	if m.ready {
		m.viewport.GotoTop()
		m.viewport.SetContent(m.renderEntries())
	}
}

func (m *Model) loadingViewMessage(serviceName string) {
	m.loading = loadingview.New(
		m.viewport.Width,
		m.viewport.Height,
		true,
		map[string]string{
			"title":   "Restarting service",
			"message": fmt.Sprintf("Restarting %s, please wait...", serviceName),
		},
	)
}
