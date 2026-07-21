// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package secretsview

import (
	"context"
	"fmt"
	"github.com/Eldara-Tech/swarmcli/docker"
	filterlist "github.com/Eldara-Tech/swarmcli/ui/components/filterable/list"
	"github.com/Eldara-Tech/swarmcli/views/confirmdialog"
	"github.com/Eldara-Tech/swarmcli/views/helpbar"
	loading "github.com/Eldara-Tech/swarmcli/views/loading"
	view "github.com/Eldara-Tech/swarmcli/views/view"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type SortField int

const (
	SortByName SortField = iota
	SortByID
	SortByUsed
	SortByCreated
	SortByUpdated
	SortByLabels
)

type Model struct {
	deps          docker.Deps
	secretsList   filterlist.FilterableList[secretItem]
	width         int
	height        int
	firstResize   bool   // tracks if we've received the first window size
	lastSnapshot  uint64 // hash of last snapshot for change detection
	polling       atomic.Bool
	visible       bool // tracks if view is currently active
	sortField     SortField
	sortAscending bool // true for ascending, false for descending

	state state
	err   error

	pendingAction     string
	confirmDialog     *confirmdialog.Model
	errorDialogActive bool
	loadingView       *loading.Model
	secrets           []docker.SecretWithDecodedData
	secretToDelete    *docker.SecretWithDecodedData

	// Create secret dialog
	createDialogActive bool
	createDialogStep   string // "source", "details-file", "details-inline"
	createDialogError  string // error message to display
	createInputFocus   int    // 0 = name, 1 = file path, 2 = labels, 3 = encode toggle
	createNameInput    textinput.Model
	createFileInput    textinput.Model // For typing file path
	createLabelsInput  textinput.Model // For typing labels (a=b,c=d)
	createSecretSource string          // "file" or "inline"
	createSecretPath   string          // selected file path from browser
	createSecretData   string
	createEncodeSecret bool // true = base64 encode the secret data
	fileBrowserActive  bool
	fileBrowserPath    string
	fileBrowserFiles   []string
	fileBrowserCursor  int

	// Used By view
	usedByViewActive bool
	usedByList       filterlist.FilterableList[usedByItem]
	usedBySecretName string

	// Spinner for slow-used-status indicator
	spinner int
}

type state int

const (
	stateLoading state = iota
	stateReady
	stateError
)

const PollInterval = 5 * time.Second
const pollTimeout = 4 * time.Second
const userActionTimeout = 15 * time.Second

func New(width, height int) *Model {
	vp := viewport.New(width, height)
	vp.SetContent("")

	m := &Model{
		width:              width,
		height:             height,
		firstResize:        true,
		state:              stateLoading,
		visible:            true,
		sortField:          SortByName,
		sortAscending:      true,
		createEncodeSecret: true,
	}

	cols := m.buildColumns()
	list := filterlist.FilterableList[secretItem]{
		Viewport: vp,
		Match: func(s secretItem, query string) bool {
			q := strings.ToLower(query)
			return strings.Contains(strings.ToLower(s.Name), q) ||
				strings.Contains(strings.ToLower(s.ID), q)
		},
		Columns: cols,
		Header: &filterlist.HeaderConfig{
			Columns: filterlist.ColumnDefs(cols),
			SortIndicator: func() (int, bool) {
				return int(m.sortField), m.sortAscending
			},
		},
		Footer: &filterlist.FooterConfig{ItemLabel: "Secret"},
	}
	list.SetOuterSize(width, height)

	// Initialize name input for create dialog
	nameInput := textinput.New()
	nameInput.Placeholder = "my-secret"
	nameInput.Prompt = "Name: "
	nameInput.CharLimit = 100
	nameInput.Width = 50

	// Initialize file path input for create dialog
	fileInput := textinput.New()
	fileInput.Placeholder = "/path/to/file"
	fileInput.Prompt = "File: "
	fileInput.CharLimit = 512
	fileInput.Width = 50

	// Initialize labels input for create dialog
	labelsInput := textinput.New()
	labelsInput.Placeholder = "key1=value1,key2=value2"
	labelsInput.Prompt = "Labels: "
	labelsInput.CharLimit = 512
	labelsInput.Width = 50

	m.secretsList = list
	m.confirmDialog = confirmdialog.New(0, 0)
	m.loadingView = loading.New(width, height, false, "Loading Docker secrets...")
	m.createNameInput = nameInput
	m.createFileInput = fileInput
	m.createLabelsInput = labelsInput

	return m
}

// CapturesInput reports whether the view is currently capturing all keyboard input.
func (m *Model) CapturesInput() bool {
	return m.createDialogActive || m.fileBrowserActive || m.confirmDialog.Visible || m.errorDialogActive
}

// IsInUsedByView returns true if the UsedBy view is currently active
func (m *Model) IsInUsedByView() bool {
	return m.usedByViewActive
}

func (m *Model) Name() string { return ViewName }

// HasActiveFilter reports whether a filter query is active.
func (m *Model) HasActiveFilter() bool {
	return m.secretsList.Query != ""
}

// IsSearching reports whether the secrets view is in a sub-view that should
// capture keys (e.g. usedBy sub-list).
func (m *Model) IsSearching() bool {
	if m.usedByViewActive {
		return true
	}
	return false
}

// ApplySearchQuery sets the filter query on the primary secrets list.
func (m *Model) ApplySearchQuery(query string) {
	m.secretsList.Query = query
	m.secretsList.ApplyFilter()
}

// ClearSearchQuery clears the filter on the primary secrets list.
func (m *Model) ClearSearchQuery() {
	m.secretsList.Query = ""
	m.secretsList.ApplyFilter()
	m.secretsList.Cursor = 0
	m.secretsList.Viewport.GotoTop()
}

func (m *Model) Init() tea.Cmd {
	l().Info("SecretsView: Init() called - starting ticker and loading secrets")
	return tea.Batch(tickCmd(), m.spinnerTickCmd(), m.loadSecretsCmd())
}

func (m *Model) spinnerTickCmd() tea.Cmd {
	return tea.Tick(80*time.Millisecond, func(t time.Time) tea.Msg {
		return SpinnerTickMsg(t)
	})
}

func tickCmd() tea.Cmd {
	return tea.Tick(PollInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	if m.usedByViewActive {
		return []helpbar.HelpEntry{
			{Key: "↑/↓", Desc: "Navigate"},
			{Key: "Enter", Desc: "Go to Stack"},
			{Key: "/", Desc: "Filter"},
			{Key: "Esc", Desc: "Back"},
		}
	}

	return []helpbar.HelpEntry{
		{Key: "↑/↓", Desc: "Navigate"},
		{Key: "n", Desc: "New"},
		{Key: "i", Desc: "Inspect"},
		{Key: "x", Desc: revealHelpDesc(), Disabled: !view.HasAction("reveal-secret")},
		{Key: "u", Desc: "Used By"},
		{Key: "ctrl+d", Desc: "Delete"},
		{Key: "?", Desc: "Help"},
		{Key: "esc", Desc: "Back"},
	}
}

func (m *Model) selectedSecret() string {
	if len(m.secretsList.Filtered) == 0 {
		return ""
	}
	return m.secretsList.Filtered[m.secretsList.Cursor].Name
}

func (m *Model) findSecretByName(name string) (*docker.SecretWithDecodedData, error) {
	for i := range m.secrets {
		if m.secrets[i].Secret.Spec.Name == name {
			return &m.secrets[i], nil
		}
	}
	return nil, fmt.Errorf("secret %q not found", name)
}

func (m *Model) addSecret(sec docker.SecretWithDecodedData) {
	m.secrets = append(m.secrets, sec)
	ctx := context.Background()
	m.secretsList.Items = append(m.secretsList.Items, secretItemFromSwarm(ctx, sec.Secret))
	m.secretsList.ApplyFilter()
}

func (m *Model) OnEnter() tea.Cmd {
	m.visible = true
	l().Info("SecretsView: OnEnter() - view is now visible")
	return m.loadSecretsCmd()
}

func (m *Model) OnExit() tea.Cmd {
	m.visible = false
	l().Info("SecretsView: OnExit() - view is no longer visible")
	return nil
}

func (m *Model) HasErrors() bool {
	return false
}

func revealHelpDesc() string {
	return view.BEHelpDesc("reveal-secret", "Reveal")
}

// validateSecretName validates a secret name
func validateSecretName(name string) error {
	if name == "" {
		return fmt.Errorf("secret name cannot be empty")
	}
	if strings.ContainsAny(name, " \t\n") {
		return fmt.Errorf("secret name cannot contain whitespace")
	}
	if strings.ContainsAny(name, "/\\:*?\"<>|") {
		return fmt.Errorf("secret name contains invalid characters")
	}
	return nil
}
