// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package stacksview

import (
	"strings"
	"swarmcli/core/primitives/hash"
	"swarmcli/docker"
	"swarmcli/views/confirmdialog"
	"swarmcli/views/helpbar"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	filterlist "swarmcli/ui/components/filterable/list"
)

type SortField int

const (
	SortByName SortField = iota
	SortByServices
	SortByTasks
	SortByError
)

type Model struct {
	List             filterlist.FilterableList[docker.StackEntry]
	Visible          bool
	nodeID           string
	ready            bool
	firstResize      bool // tracks if we've received the first window size
	width            int
	height           int
	lastSnapshot     uint64 // hash of last snapshot for change detection
	DelayInitialLoad bool   // when true, delay the first LoadStacksCmd by 3s
	sortField        SortField
	sortAscending    bool // true for ascending, false for descending
	userSetSort      bool // tracks if user manually changed sort (to avoid auto-switching)
	// expandedStacks tracks which stacks are expanded to show tasks
	expandedStacks map[string]bool
	// stackTasks caches loaded tasks per stack
	stackTasks map[string][]docker.TaskEntry
	// stackHasError marks if a stack has at least one task error (checked from snapshot)
	stackHasError map[string]bool
	// stackErrorText stores a representative error text for the stack (first found)
	stackErrorText map[string]string
	// selectedTaskIndex when navigating tasks within an expanded stack
	selectedTaskIndex int
	// errorScrollOffset for horizontal scrolling of error messages
	errorScrollOffset int
	// confirmDialog for confirming destructive actions
	confirmDialog *confirmdialog.Model
	// pendingAction tracks what action is awaiting confirmation
	pendingAction string
}

func New(width, height int) *Model {
	vp := viewport.New(width, height)
	vp.SetContent("")
	vp.YOffset = 0

	list := filterlist.FilterableList[docker.StackEntry]{
		Viewport: vp,
		// Render item will be initialized later after the column with is set
		Match: func(s docker.StackEntry, query string) bool {
			return strings.Contains(strings.ToLower(s.Name), strings.ToLower(query))
		},
	}

	return &Model{
		List:              list,
		Visible:           false,
		firstResize:       true,
		width:             width,
		height:            height,
		sortField:         SortByName,
		sortAscending:     true,
		expandedStacks:    make(map[string]bool),
		stackTasks:        make(map[string][]docker.TaskEntry),
		stackHasError:     make(map[string]bool),
		stackErrorText:    make(map[string]string),
		selectedTaskIndex: -1,
		confirmDialog:     confirmdialog.New(width, height),
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
		{Key: "p", Desc: "Tasks"},
		{Key: "ctrl+d", Desc: "Delete stack"},
		{Key: "↑/↓", Desc: "Navigate"},
		{Key: "pgup", Desc: "Page up"},
		{Key: "pgdown", Desc: "Page down"},
		{Key: "/", Desc: "Filter"},
		{Key: "?", Desc: "Help"},
		{Key: "q", Desc: "Close"},
	}
}

func LoadStacks(nodeID string) []docker.StackEntry {
	stacks, _ := LoadStacksWithErr(nodeID)
	return stacks
}

// LoadStacksWithErr refreshes snapshot and returns stack entries along with any error
func LoadStacksWithErr(nodeID string) ([]docker.StackEntry, error) {
	// Trigger a background refresh if needed, but prefer using cached data to avoid blocking UI
	docker.TriggerRefreshIfNeeded()

	snap := docker.GetSnapshot()
	if snap == nil {
		// No cached data available; attempt a synchronous refresh as a last resort
		s, err := docker.RefreshSnapshot()
		if err != nil {
			l().Errorf("LoadStacksWithErr: RefreshSnapshot failed: %v", err)
			return []docker.StackEntry{}, err
		}
		snap = s
	}
	return snap.ToStackEntries(), nil
}

func LoadStacksCmd(nodeID string) tea.Cmd {
	return func() tea.Msg {
		stacks, err := LoadStacksWithErr(nodeID)
		if err != nil {
			l().Errorf("LoadStacksCmd: Error loading stacks: %v", err)
		}

		l().Debugf("LoadStacksCmd: Loaded %v stacks", stacks)

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

		l().Debugf("CheckStacksCmd: Stacks: %+v", stacks)

		ctxName, _ := docker.GetCurrentContext()
		l().Debugf("CheckStacksCmd: docker context: %s", ctxName)

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

func (m *Model) HasActiveFilter() bool {
	return m.List.Query != ""
}

// IsSearching reports whether the list is currently in search mode.
func (m *Model) IsSearching() bool {
	return m.List.Mode == filterlist.ModeSearching
}

// HasActiveDialog reports whether a dialog is currently visible.
func (m *Model) HasActiveDialog() bool {
	return m.confirmDialog != nil && m.confirmDialog.Visible
}

// HasErrors returns true if any stack has errors
func (m *Model) HasErrors() bool {
	for _, hasErr := range m.stackHasError {
		if hasErr {
			return true
		}
	}
	return false
}
