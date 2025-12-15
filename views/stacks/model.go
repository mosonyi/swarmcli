package stacksview

import (
	"encoding/json"
	"os"
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
	List             filterlist.FilterableList[docker.StackEntry]
	Visible          bool
	nodeID           string
	ready            bool
	width            int
	height           int
	lastSnapshot     uint64 // hash of last snapshot for change detection
	DelayInitialLoad bool   // when true, delay the first LoadStacksCmd by 3s
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
	stacks, _ := LoadStacksWithErr(nodeID)
	return stacks
}

// LoadStacksWithErr refreshes snapshot and returns stack entries along with any error
func LoadStacksWithErr(nodeID string) ([]docker.StackEntry, error) {
	// Refresh the snapshot to get latest data
	snap, err := docker.RefreshSnapshot()
	if err != nil {
		l().Errorf("LoadStacksWithErr: RefreshSnapshot failed: %v", err)
		// Fall back to cached snapshot
		snap = docker.GetSnapshot()
		if snap == nil {
			return []docker.StackEntry{}, err
		}
	}
	return snap.ToStackEntries(), nil
}

func LoadStacksCmd(nodeID string) tea.Cmd {
	return func() tea.Msg {
		stacks, err := LoadStacksWithErr(nodeID)
		// Also write an initial debug dump so it's easy to confirm loading occurred
		go func() {
			type dumpEntry struct {
				Name         string `json:"name"`
				ServiceCount int    `json:"service_count"`
				NodeCount    int    `json:"node_count"`
			}
			type dump struct {
				Count   int         `json:"count"`
				Context string      `json:"context"`
				Err     string      `json:"error,omitempty"`
				Stacks  []dumpEntry `json:"stacks"`
			}
			ctxName, _ := docker.GetCurrentContext()
			errStr := ""
			if err != nil {
				errStr = err.Error()
			}
			d := dump{Count: len(stacks), Context: ctxName, Err: errStr, Stacks: []dumpEntry{}}
			for i, s := range stacks {
				if i >= 50 {
					break
				}
				d.Stacks = append(d.Stacks, dumpEntry{Name: s.Name, ServiceCount: s.ServiceCount, NodeCount: s.NodeCount})
			}
			b, _ := json.MarshalIndent(d, "", "  ")
			_ = os.WriteFile("/tmp/swarmcli_stacks_initial.json", b, 0644)
		}()

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
			// Keep polling on error instead of returning nil which would stop the tick loop
			return tickCmd()
		}

		l().Infof("CheckStacksCmd: lastHash=%s, newHash=%s, stackCount=%d",
			hash.Fmt(lastHash), hash.Fmt(newHash), len(stacks))

		// Write a debug dump (like nodes view) for troubleshooting
		go func() {
			type dumpEntry struct {
				Name         string `json:"name"`
				ServiceCount int    `json:"service_count"`
				NodeCount    int    `json:"node_count"`
			}
			type dump struct {
				Last    string      `json:"last"`
				New     string      `json:"new"`
				Count   int         `json:"count"`
				Context string      `json:"context"`
				Stacks  []dumpEntry `json:"stacks"`
			}
			ctxName, _ := docker.GetCurrentContext()
			d := dump{Last: hash.Fmt(lastHash), New: hash.Fmt(newHash), Count: len(stacks), Context: ctxName, Stacks: []dumpEntry{}}
			for i, s := range stacks {
				if i >= 50 {
					break
				}
				d.Stacks = append(d.Stacks, dumpEntry{Name: s.Name, ServiceCount: s.ServiceCount, NodeCount: s.NodeCount})
			}
			b, _ := json.MarshalIndent(d, "", "  ")
			_ = os.WriteFile("/tmp/swarmcli_stacks_check.json", b, 0644)
		}()

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
