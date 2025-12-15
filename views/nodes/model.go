package nodesview

import (
	"encoding/json"
	"os"
	"strings"
	"swarmcli/core/primitives/hash"
	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/helpbar"
	"time"

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
	lastSnapshot uint64 // Hash of last node state for change detection
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
	return tickCmd()
}

func tickCmd() tea.Cmd {
	return tea.Tick(PollInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
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
		l().Errorf("LoadNodes: RefreshSnapshot failed: %v", err)
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
func CheckNodesCmd(lastHash uint64) tea.Cmd {
	return func() tea.Msg {
		l().Info("CheckNodesCmd: Polling for node changes")

		entries := LoadNodes()
		newHash, err := hash.Compute(entries)
		if err != nil {
			l().Errorf("CheckNodesCmd: Compute hash failed: %v", err)
			return tickCmd()
		}

		l().Infof("CheckNodesCmd: lastHash=%s, newHash=%s, nodeCount=%d",
			hash.Fmt(lastHash), hash.Fmt(newHash), len(entries))

		// Write debug dump to /tmp for quick inspection during troubleshooting
		go func() {
			type dump struct {
				Last    string              `json:"last"`
				New     string              `json:"new"`
				Count   int                 `json:"count"`
				Entries []map[string]string `json:"entries"`
			}
			d := dump{Last: hash.Fmt(lastHash), New: hash.Fmt(newHash), Count: len(entries), Entries: []map[string]string{}}
			for i, e := range entries {
				if i >= 10 {
					break
				}
				labels := ""
				if e.Labels != nil {
					// Marshal labels into JSON string for readability
					lb, _ := json.Marshal(e.Labels)
					labels = string(lb)
				}
				d.Entries = append(d.Entries, map[string]string{"ID": e.ID, "Hostname": e.Hostname, "Labels": labels})
			}
			b, _ := json.MarshalIndent(d, "", "  ")
			_ = os.WriteFile("/tmp/swarmcli_nodes_check.json", b, 0644)
		}()

		// Only return update message if something changed
		if newHash != lastHash {
			l().Info("CheckNodesCmd: Change detected! Refreshing node list")
			return Msg{Entries: entries}
		}

		l().Info("CheckNodesCmd: No changes detected, scheduling next poll")
		return tickCmd()
	}
}

func (m *Model) OnEnter() tea.Cmd {
	return nil
}

func (m *Model) OnExit() tea.Cmd {
	return nil
}
