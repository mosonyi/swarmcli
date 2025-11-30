package nodesview

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
	swarmlog "swarmcli/utils/log"
	"swarmcli/views/helpbar"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	List         filterlist.FilterableList[docker.NodeEntry]
	Visible      bool
	ready        bool
	width        int
	height       int
	colWidths    map[string]int
	lastSnapshot string // Hash of last node state for change detection
}

func New(width, height int) *Model {
	vp := viewport.New(width, height)
	vp.SetContent("")

	list := filterlist.FilterableList[docker.NodeEntry]{
		Viewport: vp,
		Match: func(n docker.NodeEntry, query string) bool {
			return strings.Contains(strings.ToLower(n.Hostname), strings.ToLower(query))
		},
	}

	return &Model{
		List:    list,
		Visible: false,
		width:   width,
		height:  height,
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

// computeNodesHash creates a hash of node states for change detection
func computeNodesHash(entries []docker.NodeEntry) string {
	// Create a minimal representation focusing on key fields that indicate changes
	type nodeState struct {
		ID       string
		Hostname string
		Role     string
		State    string
		Manager  bool
		Addr     string
		Labels   map[string]string
	}
	
	states := make([]nodeState, len(entries))
	for i, e := range entries {
		states[i] = nodeState{
			ID:       e.ID,
			Hostname: e.Hostname,
			Role:     e.Role,
			State:    e.State,
			Manager:  e.Manager,
			Addr:     e.Addr,
			Labels:   e.Labels,
		}
	}
	
	data, _ := json.Marshal(states)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

func (m *Model) Name() string {
	return ViewName
}

func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "i", Desc: "Inspect"},
		{Key: "p", Desc: "ps"},
		{Key: "↑/↓", Desc: "Navigate"},
		{Key: "q", Desc: "Close"},
	}
}

func LoadNodes() []docker.NodeEntry {
	// Refresh the snapshot to get latest data
	snapshot, err := docker.RefreshSnapshot()
	if err != nil {
		logger := swarmlog.L()
		logger.Errorf("LoadNodes: RefreshSnapshot failed: %v", err)
		// Fall back to cached snapshot
		snapshot = docker.GetSnapshot()
	}
	return snapshot.ToNodeEntries()
}

func LoadNodesCmd() tea.Cmd {
	return func() tea.Msg {
		entries := LoadNodes()
		return Msg{Entries: entries}
	}
}

// CheckNodesCmd checks if nodes have changed and returns update message if so
func CheckNodesCmd(lastHash string) tea.Cmd {
	return func() tea.Msg {
		logger := swarmlog.L()
		logger.Info("CheckNodesCmd: Polling for node changes")
		
		entries := LoadNodes()
		newHash := computeNodesHash(entries)
		
		logger.Infof("CheckNodesCmd: lastHash=%s, newHash=%s, nodeCount=%d", 
			lastHash[:8], newHash[:8], len(entries))
		
		// Only return update message if something changed
		if newHash != lastHash {
			logger.Info("CheckNodesCmd: Change detected! Refreshing node list")
			return Msg{Entries: entries}
		}
		
		logger.Info("CheckNodesCmd: No changes detected, scheduling next poll")
		// Schedule next poll in 5 seconds
		return tea.Tick(PollInterval, func(t time.Time) tea.Msg {
			return TickMsg(t)
		})()
	}
}

func (m *Model) OnEnter() tea.Cmd {
	return nil
}

func (m *Model) OnExit() tea.Cmd {
	return nil
}
