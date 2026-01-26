// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package nodesview

import (
	"strings"
	"swarmcli/core/primitives/hash"
	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/confirmdialog"
	"swarmcli/views/helpbar"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type SortField int

const (
	SortByHostname SortField = iota
	SortByState
	SortByAvailability
	SortByRole
	SortByVersion
	SortByAddress
	SortByLabels
)

type Model struct {
	List                  filterlist.FilterableList[docker.NodeEntry]
	Visible               bool
	ready                 bool
	firstResize           bool // tracks if we've received the first window size
	width                 int
	height                int
	sortField             SortField
	sortAscending         bool // true for ascending, false for descending
	colWidths             map[string]int
	lastSnapshot          uint64 // Hash of last node state for change detection
	confirmDialog         *confirmdialog.Model
	errorDialogActive     bool
	labelsScrollOffset    int      // Horizontal scroll offset for labels column
	availabilityDialog    bool     // Whether availability selection dialog is visible
	availabilityNodeID    string   // Node ID for availability change
	availabilitySelection int      // Currently selected option (0=active, 1=pause, 2=drain)
	labelInputDialog      bool     // Whether label input dialog is visible
	labelInputNodeID      string   // Node ID for label add
	labelInputValue       string   // Current input value for label (key=value format)
	labelRemoveDialog     bool     // Whether label remove dialog is visible
	labelRemoveNodeID     string   // Node ID for label remove
	labelRemoveSelection  int      // Currently selected label to remove
	labelRemoveLabels     []string // List of "key=value" strings
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
		List:          list,
		Visible:       false,
		firstResize:   true,
		width:         width,
		height:        height,
		confirmDialog: confirmdialog.New(width, height),
		sortField:     SortByHostname,
		sortAscending: true,
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
		{Key: "a", Desc: "Availability"},
		{Key: "Ctrl+L", Desc: "Add label"},
		{Key: "Ctrl+R", Desc: "Remove label"},
		{Key: "Ctrl+T", Desc: "Demote node"},
		{Key: "Ctrl+O", Desc: "Promote node"},
		{Key: "Ctrl+D", Desc: "Remove node"},
		{Key: "↑/↓", Desc: "Navigate"},
		{Key: "?", Desc: "Help"},
		{Key: "q", Desc: "Close"},
	}
}

// HasActiveDialog reports whether a dialog is currently visible.
func (m *Model) HasActiveDialog() bool {
	return m.confirmDialog.Visible || m.errorDialogActive || m.availabilityDialog || m.labelInputDialog || m.labelRemoveDialog
}

func LoadNodes() []docker.NodeEntry {
	// Prefer cached snapshot to avoid blocking the UI. Trigger an async refresh if needed.
	docker.TriggerRefreshIfNeeded()

	snap := docker.GetSnapshot()
	if snap == nil {
		// Try synchronous refresh as a last resort
		s, err := docker.RefreshSnapshot()
		if err != nil {
			l().Errorf("LoadNodes: RefreshSnapshot failed: %v", err)
			return []docker.NodeEntry{}
		}
		snap = s
	}
	return snap.ToNodeEntries()
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

		l().Debugf("CheckNodesCmd: Node entries: %+v", entries)

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

// HasActiveFilter reports whether a filter query is active.
func (m *Model) HasActiveFilter() bool {
	return m.List.Query != ""
}

// IsSearching reports whether the list is currently in search mode.
func (m *Model) IsSearching() bool {
	return m.List.Mode == filterlist.ModeSearching
}
