package nodeservicesview

import (
	"context"
	"fmt"
	"swarmcli/docker"
	"swarmcli/views/confirmdialog"
	"swarmcli/views/helpbar"
	loadingview "swarmcli/views/loading"
	"time"

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

func restartServiceCmd(serviceName string, filterType FilterType, nodeID, stackName string) tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		if err := docker.RestartServiceAndWait(ctx, serviceName); err != nil {
			return fmt.Errorf("failed to restart service %s: %v", serviceName, err)
		}

		docker.RefreshSnapshot()

		var entries []ServiceEntry
		title := ""

		switch filterType {
		case NodeFilter:
			entries = LoadNodeServices(nodeID)
			title = "Node Services"
		case StackFilter:
			entries = LoadStackServices(stackName)
			title = "Stack Services"
		}

		return Msg{
			Title:      title,
			Entries:    entries,
			FilterType: filterType,
			NodeID:     nodeID,
			StackName:  stackName,
		}
	}
}
