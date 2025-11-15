package logsview

import (
	"context"
	"swarmcli/views/helpbar"
	"sync"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

// Model holds the state for the streaming logs view.
type Model struct {
	viewport      viewport.Model
	Visible       bool
	mode          string // "normal" or "search"
	searchTerm    string
	searchIndex   int
	searchMatches []int
	lines         []string // bounded: only last MaxLines kept
	MaxLines      int
	ready         bool

	// streaming control
	StreamCtx    context.Context
	StreamCancel context.CancelFunc // cancel context for streaming goroutine
	streamMu     sync.Mutex         // protects below
	streamActive bool               // whether a stream is active

	// read pump channels (internal to tea)
	linesChan chan string
	errChan   chan error

	// sync for lines slice
	mu sync.Mutex

	// follow behavior
	follow bool
}

// New creates a logs model with sensible defaults.
func New(width, height int, maxLines int) *Model {
	vp := viewport.New(width, height)
	vp.SetContent("")
	ctx, cancel := context.WithCancel(context.Background())
	return &Model{
		viewport:     vp,
		Visible:      false,
		mode:         "normal",
		lines:        make([]string, 0, 1024),
		MaxLines:     maxLines,
		StreamCtx:    ctx,
		StreamCancel: cancel,
		linesChan:    nil,
		errChan:      nil,
		follow:       true, // auto-follow by default
	}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Name() string { return ViewName }

func (m *Model) setFollow(f bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.follow = f
}

// ShortHelpItems stays compatible with your helpbar interface.
func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	if m.mode == "search" {
		return []helpbar.HelpEntry{
			{Key: "enter", Desc: "confirm"},
			{Key: "esc", Desc: "cancel"},
			{Key: "n/N", Desc: "next/prev"},
		}
	}
	return []helpbar.HelpEntry{
		{Key: "/", Desc: "search"},
		{Key: "n/N", Desc: "next/prev"},
		{Key: "f", Desc: "toggle follow"},
		{Key: "q", Desc: "close"},
	}
}

func (m *Model) OnEnter() tea.Cmd {
	// Start streaming with internal context
	return StartStreamingCmd(m.StreamCtx, m.ServiceEntry, 1000, m.MaxLines)
}

func (m *Model) OnExit() tea.Cmd {
	return m.StopStreamingCmd()
}
