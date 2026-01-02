package servicesview

import (
	"fmt"
	"strings"
	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/confirmdialog"
	"swarmcli/views/helpbar"
	loadingview "swarmcli/views/loading"
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
		firstResize:   true,
		width:         width,
		height:        height,
		confirmDialog: confirmdialog.New(width, height),
		loading:       ld,
		msgCh:         make(chan tea.Msg),
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
