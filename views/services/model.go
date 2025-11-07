package servicesview

import (
	"fmt"
	"log"
	swarmlog "swarmcli/utils/log"
	"swarmcli/views/confirmdialog"
	"swarmcli/views/helpbar"
	loadingview "swarmcli/views/loading"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"
)

type FilterType int

const (
	NodeFilter FilterType = iota
	StackFilter
	AllFilter
)

func l() *zap.SugaredLogger {
	return swarmlog.Logger.With("view", "services")
}

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

	msgCh chan tea.Msg

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
		msgCh:         make(chan tea.Msg, 10),
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

func sendMsg(ch chan tea.Msg, msg tea.Msg) {
	select {
	case ch <- msg:
	default:
		log.Println("[sendMsg] msg channel full, dropping message")
	}
}

func (m Model) listenForMessages() tea.Cmd {
	if m.msgCh == nil {
		log.Println("[listenForMessages] no message channel, skipping")
		return nil
	}

	return func() tea.Msg {
		msg, ok := <-m.msgCh
		if !ok {
			log.Println("[listenForMessages] channel closed — stopping listener")
			return nil
		}
		return msg
	}
}
