package stacksview

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"
	"swarmcli/docker"
	swarmlog "swarmcli/utils/log"
	"swarmcli/views/helpbar"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	filterlist "swarmcli/ui/components/filterable/list"
)

type Model struct {
	List         filterlist.FilterableList[docker.StackEntry]
	Visible      bool
	nodeID       string
	ready        bool
	width        int
	height       int
	lastSnapshot string // hash of last snapshot for change detection
}

func New(width, height int) *Model {
	vp := viewport.New(width, height)
	vp.SetContent("")

	list := filterlist.FilterableList[docker.StackEntry]{
		Viewport: vp,
		// Render item will be initialized later after the column with is set
		Match: func(s docker.StackEntry, query string) bool {
			return strings.Contains(strings.ToLower(s.Name), strings.ToLower(query))
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

// computeStacksHash creates a hash of stack states for change detection
func computeStacksHash(entries []docker.StackEntry) string {
	type stackState struct {
		Name         string
		ServiceCount int
		NodeCount    int
	}

	states := make([]stackState, len(entries))
	for i, e := range entries {
		states[i] = stackState{
			Name:         e.Name,
			ServiceCount: e.ServiceCount,
			NodeCount:    e.NodeCount,
		}
	}

	data, _ := json.Marshal(states)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
}

func (m *Model) Name() string { return ViewName }

func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "i/enter", Desc: "Services"},
		{Key: "↑/↓", Desc: "Navigate"},
		{Key: "pgup", Desc: "Page up"},
		{Key: "pgdown", Desc: "Page down"},
		{Key: "/", Desc: "Filter"},
		{Key: "q", Desc: "Close"},
	}
}

func LoadStacks(nodeID string) []docker.StackEntry {
	// Refresh the snapshot to get latest data
	snap, err := docker.RefreshSnapshot()
	if err != nil {
		logger := swarmlog.L()
		logger.Errorf("LoadStacks: RefreshSnapshot failed: %v", err)
		// Fall back to cached snapshot
		snap = docker.GetSnapshot()
	}
	return snap.ToStackEntries()
}

func LoadStacksCmd(nodeID string) tea.Cmd {
	return func() tea.Msg {
		stacks := LoadStacks(nodeID)
		return Msg{NodeID: nodeID, Stacks: stacks}
	}
}

// CheckStacksCmd checks if stacks have changed and returns update message if so
func CheckStacksCmd(lastHash string, nodeID string) tea.Cmd {
	return func() tea.Msg {
		logger := swarmlog.L()
		logger.Info("CheckStacksCmd: Polling for stack changes")

		stacks := LoadStacks(nodeID)
		newHash := computeStacksHash(stacks)

		logger.Infof("CheckStacksCmd: lastHash=%s, newHash=%s, stackCount=%d",
			lastHash[:8], newHash[:8], len(stacks))

		// Only return update message if something changed
		if newHash != lastHash {
			logger.Info("CheckStacksCmd: Change detected! Refreshing stack list")
			return Msg{NodeID: nodeID, Stacks: stacks}
		}

		logger.Info("CheckStacksCmd: No changes detected, scheduling next poll")
		// Schedule next poll in 5 seconds
		return tea.Tick(PollInterval, func(t time.Time) tea.Msg {
			return TickMsg(t)
		})()
	}
}

func LoadStacksOld(nodeID string) tea.Cmd {
	return func() tea.Msg {
		snap, err := docker.GetOrRefreshSnapshot()
		if err != nil {
			return Msg{NodeID: nodeID, Stacks: nil}
		}
		stacks := snap.ToStackEntries()
		return Msg{NodeID: nodeID, Stacks: stacks}
	}
}

func (m *Model) OnEnter() tea.Cmd { return nil }
func (m *Model) OnExit() tea.Cmd  { return nil }
