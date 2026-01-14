package secretsview

import (
	"context"
	"fmt"
	"strings"
	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/confirmdialog"
	"swarmcli/views/helpbar"
	loading "swarmcli/views/loading"
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
	secretsList   filterlist.FilterableList[secretItem]
	width         int
	height        int
	firstResize   bool   // tracks if we've received the first window size
	lastSnapshot  uint64 // hash of last snapshot for change detection
	visible       bool   // tracks if view is currently active
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

	// Reveal secret dialog
	revealDialogActive  bool
	revealSecretName    string
	revealContent       string
	revealDecoded       bool // true if content was base64 decoded
	revealViewport      viewport.Model
	revealingInProgress bool // true while waiting for secret to be revealed

	// Cached column widths for header alignment
	colNameWidth int
	colIdWidth   int

	// Spinner for slow-used-status indicator
	spinner int

	// Horizontal scroll offset for labels column
	labelsScrollOffset int
}

type state int

const (
	stateLoading state = iota
	stateReady
	stateError
)

const PollInterval = 5 * time.Second

func New(width, height int) *Model {
	vp := viewport.New(width, height)
	vp.SetContent("")

	list := filterlist.FilterableList[secretItem]{
		Viewport: vp,
		Match: func(s secretItem, query string) bool {
			q := strings.ToLower(query)
			return strings.Contains(strings.ToLower(s.Name), q) ||
				strings.Contains(strings.ToLower(s.ID), q)
		},
	}

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

	// Initialize viewport for reveal dialog - use full dimensions like inspect
	revealVp := viewport.New(width, height)

	return &Model{
		secretsList:        list,
		width:              width,
		height:             height,
		firstResize:        true,
		state:              stateLoading,
		visible:            true,
		confirmDialog:      confirmdialog.New(0, 0),
		loadingView:        loading.New(width, height, false, "Loading Docker secrets..."),
		createNameInput:    nameInput,
		createFileInput:    fileInput,
		createLabelsInput:  labelsInput,
		createEncodeSecret: true, // Default to encoding
		revealViewport:     revealVp,
		sortField:          SortByName,
		sortAscending:      true,
	}
}

// HasActiveDialog returns true if any dialog is currently active
func (m *Model) HasActiveDialog() bool {
	return m.revealDialogActive || m.createDialogActive || m.fileBrowserActive || m.confirmDialog.Visible || m.errorDialogActive
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

// IsSearching reports whether the secrets or UsedBy list is in search mode.
func (m *Model) IsSearching() bool {
	if m.usedByViewActive {
		return m.usedByList.Mode == filterlist.ModeSearching
	}
	return m.secretsList.Mode == filterlist.ModeSearching
}

func (m *Model) Init() tea.Cmd {
	l().Info("SecretsView: Init() called - starting ticker and loading secrets")
	return tea.Batch(tickCmd(), m.spinnerTickCmd(), LoadSecrets())
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

func LoadSecrets() tea.Cmd {
	return loadSecretsCmd()
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
		{Key: "x", Desc: "Reveal"},
		{Key: "u", Desc: "Used By"},
		{Key: "ctrl+d", Desc: "Delete"},
		{Key: "?", Desc: "Help"},
		{Key: "esc/q", Desc: "Back"},
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
	return LoadSecrets()
}

func (m *Model) OnExit() tea.Cmd {
	m.visible = false
	l().Info("SecretsView: OnExit() - view is no longer visible")
	return nil
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
