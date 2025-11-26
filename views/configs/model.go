package configsview

import (
	"fmt"
	"strings"
	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/confirmdialog"
	"swarmcli/views/helpbar"
	loading "swarmcli/views/loading"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	configsList filterlist.FilterableList[configItem]
	state       state
	err         error

	pendingAction      string
	confirmDialog      *confirmdialog.Model
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
	fl := filterlist.FilterableList[configItem]{
		Viewport: viewport.Model{
			Width:  width,
			Height: height,
		},
		RenderItem: func(item configItem, selected bool, colWidth int) string {
			nameCol := fmt.Sprintf("%-*s", colWidth, item.Name) // left-align name
			line := fmt.Sprintf("%s  ID: %s", nameCol, item.ID)
			if selected {
				return "> " + line
			}
			return "  " + line
		},
		Match: func(item configItem, query string) bool {
			// match by name or ID
			q := strings.ToLower(query)
			return strings.Contains(strings.ToLower(item.Name), q) ||
				strings.Contains(strings.ToLower(item.ID), q)
		},
	}

	fl.ComputeAndSetColWidth(func(item configItem) string { return item.Name }, 10)

	return &Model{
		configsList:   fl,
		state:         stateLoading,
		confirmDialog: confirmdialog.New(0, 0),
		loadingView:   loading.New(width, height, false, "Loading Docker configs..."),
	}
}

func (m *Model) Name() string { return ViewName }

func (m *Model) Init() tea.Cmd {
	return nil
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
		{Key: "r", Desc: "rotate"},
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
	return nil
}

func (m *Model) OnExit() tea.Cmd {
	return nil
}
