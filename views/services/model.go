package servicesview

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
	swarmlog "swarmcli/utils/log"
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
	AllFilter
)

func l() *swarmlog.SwarmLogger {
	return swarmlog.L().With("docker", "client")
}

type Model struct {
	List         filterlist.FilterableList[docker.ServiceEntry]
	Visible      bool
	title        string
	ready        bool
	width        int
	height       int
	lastSnapshot string // hash of last snapshot for change detection

	// Filter
	filterType FilterType
	nodeID     string
	stackName  string

	msgCh chan tea.Msg

	confirmDialog *confirmdialog.Model
	loading       *loadingview.Model
}

func New(width, height int) *Model {
	vp := viewport.New(width, height)
	ld := loadingview.New(width, height, false, "Please wait...")

	list := filterlist.FilterableList[docker.ServiceEntry]{
		Viewport: vp,
		Match: func(s docker.ServiceEntry, query string) bool {
			return strings.Contains(strings.ToLower(s.ServiceName), strings.ToLower(query))
		},
	}

	return &Model{
		List:          list,
		Visible:       false,
		width:         width,
		height:        height,
		confirmDialog: confirmdialog.New(width, height),
		loading:       ld,
		msgCh:         make(chan tea.Msg),
	}
}

func (m *Model) Init() tea.Cmd {
	return m.tickCmd()
}

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(PollInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// computeServicesHash creates a hash of service states for change detection
func computeServicesHash(entries []docker.ServiceEntry) string {
	type serviceState struct {
		StackName      string
		ServiceName    string
		ServiceID      string
		ReplicasOnNode int
		ReplicasTotal  int
	}
	
	states := make([]serviceState, len(entries))
	for i, e := range entries {
		states[i] = serviceState{
			StackName:      e.StackName,
			ServiceName:    e.ServiceName,
			ServiceID:      e.ServiceID,
			ReplicasOnNode: e.ReplicasOnNode,
			ReplicasTotal:  e.ReplicasTotal,
		}
	}
	
	data, _ := json.Marshal(states)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

func (m *Model) Name() string { return ViewName }

func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "i", Desc: "Inspect"},
		{Key: "↑/↓", Desc: "Navigate"},
		{Key: "r", Desc: "Restart service"},
		{Key: "l", Desc: "View logs"},
		{Key: "q", Desc: "Close"},
	}
}

func (m *Model) loadingViewMessage(serviceName string) {
	m.loading = loadingview.New(
		m.List.Viewport.Width,
		m.List.Viewport.Height,
		true,
		map[string]string{
			"title":   "Restarting service",
			"message": fmt.Sprintf("Restarting %s, please wait...", serviceName),
		},
	)
}

func sendMsg(ch chan tea.Msg, msg tea.Msg) {
	// Block briefly until UI consumes, to avoid drop storm at the end
	select {
	case ch <- msg:
	case <-time.After(200 * time.Millisecond):
		l().Infof("[sendMsg] timeout waiting to deliver progress update")
	}
}

func (m *Model) listenForMessages() tea.Cmd {
	if m.msgCh == nil {
		l().Debugf("[listenForMessages] no message channel, skipping")
		return nil
	}

	return func() tea.Msg {
		msg, ok := <-m.msgCh
		if !ok {
			l().Debugf("[listenForMessages] channel closed — stopping listener")
			return nil
		}
		return msg
	}
}

func (m *Model) OnEnter() tea.Cmd {
	return nil
}

func (m *Model) OnExit() tea.Cmd {
	return nil
}
