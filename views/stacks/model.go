package stacksview

import (
	"strings"
	"swarmcli/core/primitives/hash"
	"swarmcli/docker"
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
	lastSnapshot uint64 // hash of last snapshot for change detection
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
		l().Errorf("LoadStacks: RefreshSnapshot failed: %v", err)
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
func CheckStacksCmd(lastHash uint64, nodeID string) tea.Cmd {
	return func() tea.Msg {
		l().Info("CheckStacksCmd: Polling for stack changes")

		stacks := LoadStacks(nodeID)
		var err error
		newHash, err := hash.Compute(stacks)
		if err != nil {
			l().Errorf("CheckStacksCmd: Error computing hash: %v", err)
			return nil
		}

		l().Infof("CheckStacksCmd: lastHash=%s, newHash=%s, stackCount=%d",
			hash.Fmt(lastHash), hash.Fmt(newHash), len(stacks))

		// Only return update message if something changed
		if newHash != lastHash {
			l().Info("CheckStacksCmd: Change detected! Refreshing stack list")
			return Msg{NodeID: nodeID, Stacks: stacks}
		}

		l().Info("CheckStacksCmd: No changes detected, scheduling next poll")
		return tickCmd()
	}
}

func (m *Model) OnEnter() tea.Cmd { return nil }
func (m *Model) OnExit() tea.Cmd  { return nil }
