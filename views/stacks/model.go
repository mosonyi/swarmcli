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

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	filterlist "swarmcli/ui/components/filterable/list"
)

const defaultStackTemplate = `version: '3.8'

services:
  # Define your services here
  # Example:
  # myservice:
  #   image: myimage:latest
  #   ports:
  #     - "8080:8080"
  #   environment:
  #     - MY_VAR=value

networks:
  # Define custom networks if needed
`

type SortField int

const (
	SortByName SortField = iota
	SortByServices
	SortByTasks
	SortByError
)

type Model struct {
	deps             docker.Deps
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

	// Create stack dialog
	createDialogActive  bool
	createDialogStep    string // "source", "details-file", "details-inline"
	createDialogError   string // error message to display
	createInputFocus    int    // 0 = name, 1 = file path/content
	createNameInput     textinput.Model
	createFileInput     textinput.Model // For typing file path
	createStackSource   string          // "file" or "inline"
	createStackPath     string          // selected file path from browser
	createDialogContent string          // YAML content for the stack
	fileBrowserActive   bool
	fileBrowserPath     string
	fileBrowserFiles    []string
	fileBrowserCursor   int

	// Edit stack tracking
	editStackName string // non-empty when editing a stack (vs creating new)
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

	// Initialize name input for create dialog
	nameInput := textinput.New()
	nameInput.Placeholder = "my-stack"
	nameInput.Prompt = "Stack Name: "
	nameInput.CharLimit = 100
	nameInput.Width = 50

	// Initialize file path input for create dialog
	fileInput := textinput.New()
	fileInput.Placeholder = "/path/to/compose.yml"
	fileInput.Prompt = "Compose File: "
	fileInput.CharLimit = 512
	fileInput.Width = 50

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
		createNameInput:   nameInput,
		createFileInput:   fileInput,
		createDialogStep:  "source",
		createStackSource: "file",
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
		{Key: "n", Desc: "New stack"},
		{Key: "e", Desc: "Edit"},
		{Key: "i/enter", Desc: "Services"},
		{Key: "d", Desc: "Inspect"},
		{Key: "p", Desc: "Tasks"},
		{Key: "ctrl+d", Desc: "Delete"},
		{Key: "↑/↓", Desc: "Navigate"},
		{Key: "/", Desc: "Filter"},
		{Key: "?", Desc: "Help"},
		{Key: "q", Desc: "Close"},
	}
}

func (m *Model) LoadStacksCmd(nodeID string) tea.Cmd {
	snapOps := m.deps.Snapshot
	return func() tea.Msg {
		snapOps.TriggerRefreshIfNeeded()

		snap := snapOps.GetSnapshot()
		if snap == nil {
			s, err := snapOps.RefreshSnapshot()
			if err != nil {
				l().Errorf("LoadStacksCmd: RefreshSnapshot failed: %v", err)
				return Msg{NodeID: nodeID, Stacks: []docker.StackEntry{}}
			}
			snap = s
		}
		stacks := snap.ToStackEntries()

		l().Debugf("LoadStacksCmd: Loaded %v stacks", stacks)

		return Msg{NodeID: nodeID, Stacks: stacks}
	}
}

// checkStacksCmd checks if stacks have changed and returns update message if so
func (m *Model) checkStacksCmd(lastHash uint64, nodeID string) tea.Cmd {
	snapOps := m.deps.Snapshot
	clusterOps := m.deps.ClusterInfo
	return func() tea.Msg {
		l().Info("checkStacksCmd: Polling for stack changes")

		snapOps.TriggerRefreshIfNeeded()

		snap := snapOps.GetSnapshot()
		if snap == nil {
			s, err := snapOps.RefreshSnapshot()
			if err != nil {
				l().Errorf("checkStacksCmd: RefreshSnapshot failed: %v", err)
				return tickCmd()
			}
			snap = s
		}
		stacks := snap.ToStackEntries()

		newHash, err := hash.Compute(stacks)
		if err != nil {
			l().Errorf("checkStacksCmd: Error computing hash: %v", err)
			// Keep polling on error instead of returning nil which would stop the tick loop
			return tickCmd()
		}

		l().Infof("checkStacksCmd: lastHash=%s, newHash=%s, stackCount=%d",
			hash.Fmt(lastHash), hash.Fmt(newHash), len(stacks))

		l().Debugf("checkStacksCmd: Stacks: %+v", stacks)

		ctxName, _ := clusterOps.GetCurrentContext()
		l().Debugf("checkStacksCmd: docker context: %s", ctxName)

		// Only return update message if something changed
		if newHash != lastHash {
			l().Info("checkStacksCmd: Change detected! Refreshing stack list")
			return Msg{NodeID: nodeID, Stacks: stacks}
		}

		l().Info("checkStacksCmd: No changes detected, scheduling next poll")
		return tickCmd()
	}
}

func (m *Model) OnEnter() tea.Cmd { return m.LoadStacksCmd(m.nodeID) }
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
	return (m.confirmDialog != nil && m.confirmDialog.Visible) || m.createDialogActive || m.fileBrowserActive
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
