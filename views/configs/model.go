package configsview

import (
	"fmt"
	"strings"
	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/confirmdialog"
	"swarmcli/views/helpbar"
	loading "swarmcli/views/loading"
	"time"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	configsList  filterlist.FilterableList[configItem]
	width        int
	height       int
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

	return &Model{
		configsList:   list,
		width:         width,
		height:        height,
		state:         stateLoading,
		visible:       true,
		confirmDialog: confirmdialog.New(0, 0),
		loadingView:   loading.New(width, height, false, "Loading Docker configs..."),
	}
}

func (m *Model) Name() string { return ViewName }

func (m *Model) Init() tea.Cmd {
	l().Info("ConfigsView: Init() called - starting ticker and loading configs")
	return tea.Batch(tickCmd(), LoadConfigs())
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
	return []helpbar.HelpEntry{
		{Key: "↑/↓", Desc: "Navigate"},
		{Key: "i", Desc: "Inspect"},
		{Key: "Enter", Desc: "Check"},
		{Key: "e", Desc: "Edit"},
		{Key: "r", Desc: "Rotate"},
		{Key: "ctrl+d", Desc: "Delete"},
		{Key: "q", Desc: "Back"},
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
	m.configsList.Items = append(m.configsList.Items, configItemFromSwarm(cfg.Config))
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
	return m.confirmDialog.Visible || m.errorDialogActive
}
