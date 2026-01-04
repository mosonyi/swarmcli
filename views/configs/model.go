package configsview

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

type Model struct {
	configsList  filterlist.FilterableList[configItem]
	width        int
	height       int
	firstResize  bool   // tracks if we've received the first window size
	lastSnapshot uint64 // hash of last snapshot for change detection
	visible      bool   // tracks if view is currently active

	state state
	err   error

	pendingAction      string
	confirmDialog      *confirmdialog.Model
	errorDialogActive  bool
	loadingView        *loading.Model
	configs            []docker.ConfigWithDecodedData
	configToRotateFrom *docker.ConfigWithDecodedData
	configToRotateInto *docker.ConfigWithDecodedData
	configToDelete     *docker.ConfigWithDecodedData

	// Create config dialog
	createDialogActive bool
	createDialogStep   string // "source", "details-file", "details-inline"
	createDialogError  string // error message to display
	createInputFocus   int    // 0 = name, 1 = file path
	createNameInput    textinput.Model
	createFileInput    textinput.Model // For typing file path
	createConfigSource string          // "file" or "inline"
	createConfigPath   string          // selected file path from browser
	createConfigData   string
	fileBrowserActive  bool
	fileBrowserPath    string
	fileBrowserFiles   []string
	fileBrowserCursor  int

	// Used By view
	usedByViewActive bool
	usedByList       filterlist.FilterableList[usedByItem]
	usedByConfigName string

	// Cached column widths for header alignment
	colNameWidth int
	colIdWidth   int

	// Spinner for slow-used-status indicator
	spinner int
}

type state int

const (
	stateLoading state = iota
	stateReady
	stateError
)

func New(width, height int) *Model {
	vp := viewport.New(width, height)
	vp.SetContent("")

	list := filterlist.FilterableList[configItem]{
		Viewport: vp,
		Match: func(c configItem, query string) bool {
			q := strings.ToLower(query)
			return strings.Contains(strings.ToLower(c.Name), q) ||
				strings.Contains(strings.ToLower(c.ID), q)
		},
	}

	// Initialize name input for create dialog
	nameInput := textinput.New()
	nameInput.Placeholder = "my-config"
	nameInput.Prompt = "Name: "
	nameInput.CharLimit = 100
	nameInput.Width = 50

	// Initialize file path input for create dialog
	fileInput := textinput.New()
	fileInput.Placeholder = "/path/to/file"
	fileInput.Prompt = "File: "
	fileInput.CharLimit = 512
	fileInput.Width = 50

	return &Model{
		configsList:     list,
		width:           width,
		height:          height,
		firstResize:     true,
		state:           stateLoading,
		visible:         true,
		confirmDialog:   confirmdialog.New(0, 0),
		loadingView:     loading.New(width, height, false, "Loading Docker configs..."),
		createNameInput: nameInput,
		createFileInput: fileInput,
	}
}

func (m *Model) Name() string { return ViewName }

// HasActiveFilter reports whether a filter query is active.
func (m *Model) HasActiveFilter() bool {
	return m.configsList.Query != ""
}

// IsSearching reports whether the configs or UsedBy list is in search mode.
func (m *Model) IsSearching() bool {
	if m.usedByViewActive {
		return m.usedByList.Mode == filterlist.ModeSearching
	}
	return m.configsList.Mode == filterlist.ModeSearching
}

func (m *Model) Init() tea.Cmd {
	l().Info("ConfigsView: Init() called - starting ticker and loading configs")
	return tea.Batch(tickCmd(), m.spinnerTickCmd(), LoadConfigs())
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

func LoadConfigs() tea.Cmd {
	return loadConfigsCmd()
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
		{Key: "c", Desc: "Clone"},
		{Key: "i", Desc: "Inspect"},
		{Key: "u", Desc: "Used By"},
		{Key: "Enter", Desc: "Check"},
		{Key: "e", Desc: "Edit & Rotate"},
		{Key: "ctrl+d", Desc: "Delete"},
		{Key: "esc/q", Desc: "Back"},
	}
}

func (m *Model) selectedConfig() string {
	if len(m.configsList.Filtered) == 0 {
		return ""
	}
	return m.configsList.Filtered[m.configsList.Cursor].Name
}

func (m *Model) findConfigByName(name string) (*docker.ConfigWithDecodedData, error) {
	for i := range m.configs {
		if m.configs[i].Config.Spec.Name == name {
			return &m.configs[i], nil
		}
	}
	return nil, fmt.Errorf("config %q not found", name)
}

func (m *Model) addConfig(cfg docker.ConfigWithDecodedData) {
	m.configs = append(m.configs, cfg)
	ctx := context.Background()
	m.configsList.Items = append(m.configsList.Items, configItemFromSwarm(ctx, cfg.Config))
	m.configsList.ApplyFilter()
}

func (m *Model) OnEnter() tea.Cmd {
	m.visible = true
	l().Info("ConfigsView: OnEnter() - view is now visible")
	return LoadConfigs()
}

func (m *Model) OnExit() tea.Cmd {
	m.visible = false
	l().Info("ConfigsView: OnExit() - view is no longer visible")
	return nil
}

// HasActiveDialog returns true if a dialog is currently visible
func (m *Model) HasActiveDialog() bool {
	return m.confirmDialog.Visible || m.errorDialogActive || m.createDialogActive || m.fileBrowserActive
}

// IsInUsedByView returns true if currently viewing the used-by list
func (m *Model) IsInUsedByView() bool {
	return m.usedByViewActive
}

// validateConfigName validates a config name
func validateConfigName(name string) error {
	if name == "" {
		return fmt.Errorf("config name cannot be empty")
	}
	if strings.ContainsAny(name, " \t\n") {
		return fmt.Errorf("config name cannot contain whitespace")
	}
	if strings.ContainsAny(name, "/\\:*?\"<>|") {
		return fmt.Errorf("config name contains invalid characters")
	}
	return nil
}
