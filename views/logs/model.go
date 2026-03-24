// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

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
	deps          docker.Deps
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
	// node filter - if set, only show logs from this node
	nodeFilter string
	// app-level "/" filter — hides non-matching lines
	filterQuery string
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

// GetSearchMode is exported for app to check search mode status
func (m *Model) GetSearchMode() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.mode == "search"
}

// IsSearching is an alias for GetSearchMode for compatibility with app-level search checks
func (m *Model) IsSearching() bool {
	return m.GetSearchMode()
}

// CapturesInput returns true when the view is capturing all keyboard input.
func (m *Model) CapturesInput() bool {
	return m.getNodeSelectVisible()
}

func (m *Model) getFilterQuery() string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.filterQuery
}

// ApplySearchQuery implements view.Filterable — sets the app-level "/" filter.
func (m *Model) ApplySearchQuery(query string) {
	m.mu.Lock()
	m.filterQuery = query
	m.mu.Unlock()
	m.highlightContent()
	if m.ready && m.getFollow() {
		m.viewport.GotoBottom()
	}
}

// ClearSearchQuery implements view.Filterable — clears the app-level "/" filter.
func (m *Model) ClearSearchQuery() {
	m.mu.Lock()
	m.filterQuery = ""
	m.mu.Unlock()
	m.highlightContent()
	if m.ready && m.getFollow() {
		m.viewport.GotoBottom()
	}
}

// HasActiveFilter returns true when the app-level "/" filter is active.
func (m *Model) HasActiveFilter() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.filterQuery != ""
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
		{Key: "ctrl+f", Desc: "Search"},
		{Key: "n/N", Desc: "Next/prev"},
		{Key: "s", Desc: "Toggle AutoScroll"},
		{Key: "w", Desc: "Toggle wrap"},
		{Key: "o", Desc: "Filter node"},
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

func (m *Model) HasErrors() bool {
	return false
}

// extractUniqueNodes returns a sorted list of nodes where the service has running tasks
func (m *Model) extractUniqueNodes() []string {
	snapshotOps := m.deps.Snapshot
	snap := snapshotOps.GetSnapshot()
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
