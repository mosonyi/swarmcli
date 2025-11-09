package configsview

import (
	"swarmcli/docker"
	"swarmcli/views/confirmdialog"
	"swarmcli/views/helpbar"
	loading "swarmcli/views/loading"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	list           list.Model
	state          state
	err            error
	pendingAction  string
	confirmDialog  confirmdialog.Model
	loadingView    loading.Model
	configToRotate *docker.ConfigWithDecodedData // store edited config for rotation
}

type state int

const (
	stateLoading state = iota
	stateReady
	stateError
)

func New(width, height int) Model {
	m := Model{
		list:          list.New([]list.Item{}, itemDelegate{}, 0, 0),
		loadingView:   loading.New(width, height, false, "Loading Docker configs..."),
		state:         stateLoading,
		confirmDialog: confirmdialog.New(0, 0),
	}
	m.list.Title = "Docker Configs"
	return m
}

func (m Model) Name() string { return ViewName }

func (m Model) Init() tea.Cmd {
	return nil
}

func LoadConfigs() tea.Cmd {
	return loadConfigsCmd()
}

func (m Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "↑/↓", Desc: "Navigate"},
		{Key: "Enter/k", Desc: "Inspect"},
		{Key: "e", Desc: "Edit"},
		{Key: "r", Desc: "rotate"},
		{Key: "q", Desc: "Back"},
	}
}

func (m Model) selectedConfig() string {
	if item, ok := m.list.SelectedItem().(configItem); ok {
		return item.Name
	}
	return ""
}
