package logsview

import (
	"context"
	"sort"
	"swarmcli/docker"
	"swarmcli/views/helpbar"
	"sync"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/swarm"
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
	lineNodes     []string // node name for each line (parallel to lines)
	MaxLines      int
	ready         bool

	ServiceEntry docker.ServiceEntry

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
	// wrap behavior
	wrap bool
	// horizontal scroll offset when wrap is off
	horizontalOffset int
	// fullscreen mode
	fullscreen bool
	// node filter - if set, only show logs from this node
	nodeFilter string
	// node selection dialog
	nodeSelectVisible bool
	nodeSelectCursor  int
	nodeSelectNodes   []string
}

// New creates a logs model with sensible defaults.
func New(width, height int, maxLines int, service docker.ServiceEntry) *Model {
	vp := viewport.New(width, height)
	vp.SetContent("")
	ctx, cancel := context.WithCancel(context.Background())
	return &Model{
		viewport:          vp,
		Visible:           false,
		mode:              "normal",
		lines:             make([]string, 0, 1024),
		lineNodes:         make([]string, 0, 1024),
		MaxLines:          maxLines,
		StreamCtx:         ctx,
		StreamCancel:      cancel,
		ServiceEntry:      service,
		linesChan:         nil,
		errChan:           nil,
		follow:            true, // auto-follow by default
		wrap:              true, // wrap lines by default
		horizontalOffset:  0,
		fullscreen:        false,
		nodeFilter:        "", // empty = show all nodes
		nodeSelectVisible: false,
		nodeSelectCursor:  0,
		nodeSelectNodes:   []string{},
	}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Name() string { return ViewName }

func (m *Model) setFollow(f bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.follow = f
}

func (m *Model) getNodeFilter() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.nodeFilter
}

func (m *Model) setNodeFilter(filter string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodeFilter = filter
}

func (m *Model) getNodeSelectVisible() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.nodeSelectVisible
}

func (m *Model) setNodeSelectVisible(visible bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.nodeSelectVisible = visible
}

func (m *Model) GetNodeSelectVisible() bool {
	return m.getNodeSelectVisible()
}

func (m *Model) getFollow() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.follow
}

func (m *Model) setWrap(w bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.wrap = w
}

func (m *Model) getWrap() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.wrap
}

func (m *Model) setFullscreen(f bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.fullscreen = f
}

func (m *Model) getFullscreen() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.fullscreen
}

// GetFullscreen is exported for app to check fullscreen status
func (m *Model) GetFullscreen() bool {
	return m.getFullscreen()
}

// GetSearchMode is exported for app to check search mode status
func (m *Model) GetSearchMode() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.mode == "search"
}

// ShortHelpItems stays compatible with your helpbar interface.
func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	if m.mode == "search" {
		return []helpbar.HelpEntry{
			{Key: "enter", Desc: "Confirm"},
			{Key: "esc", Desc: "Cancel"},
			{Key: "n/N", Desc: "Next/prev"},
		}
	}

	entries := []helpbar.HelpEntry{
		{Key: "/", Desc: "Search"},
		{Key: "n/N", Desc: "Next/prev"},
		{Key: "s", Desc: "Toggle AutoScroll"},
		{Key: "w", Desc: "Toggle wrap"},
		{Key: "o", Desc: "Filter node"},
		{Key: "f", Desc: "Fullscreen"},
	}

	// Show left/right help only when wrap is off
	if !m.getWrap() {
		entries = append(entries, helpbar.HelpEntry{Key: "←/→", Desc: "Scroll"})
	}

	entries = append(entries, helpbar.HelpEntry{Key: "q", Desc: "Close"})
	return entries
}

func (m *Model) OnEnter() tea.Cmd {
	// We start streaming with the factory method
	return nil
}

func (m *Model) OnExit() tea.Cmd {
	return m.StopStreamingCmd()
}

// extractUniqueNodes returns a sorted list of nodes where the service has running tasks
func (m *Model) extractUniqueNodes() []string {
	snap := docker.GetSnapshot()
	if snap == nil {
		return []string{"All nodes"}
	}

	nodeMap := make(map[string]string) // nodeID -> hostname

	// Find all tasks for this service
	for _, task := range snap.Tasks {
		if task.ServiceID == m.ServiceEntry.ServiceID && task.DesiredState == swarm.TaskStateRunning {
			// Get the node hostname for this task
			for _, node := range snap.Nodes {
				if node.ID == task.NodeID {
					if node.Description.Hostname != "" {
						nodeMap[node.ID] = node.Description.Hostname
					}
					break
				}
			}
		}
	}

	// Convert to sorted slice of hostnames
	nodes := make([]string, 0, len(nodeMap))
	for _, hostname := range nodeMap {
		nodes = append(nodes, hostname)
	}
	sort.Strings(nodes)

	// Add "All nodes" option at the beginning
	return append([]string{"All nodes"}, nodes...)
}
