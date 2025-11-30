package configsview

import (
	"crypto/sha256"
	"encoding/json"
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
	lastSnapshot string // hash of last snapshot for change detection

	state state
	err   error

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
		confirmDialog: confirmdialog.New(0, 0),
		loadingView:   loading.New(width, height, false, "Loading Docker configs..."),
	}
}

func (m *Model) Name() string { return ViewName }

func (m *Model) Init() tea.Cmd {
	return m.tickCmd()
}

func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(PollInterval, func(t time.Time) tea.Msg {
		return TickMsg(t)
	})
}

// computeConfigsHash creates a hash of config states for change detection
func computeConfigsHash(configs []docker.ConfigWithDecodedData) string {
	type configState struct {
		ID      string
		Name    string
		Version uint64
	}

	states := make([]configState, len(configs))
	for i, c := range configs {
		states[i] = configState{
			ID:      c.Config.ID,
			Name:    c.Config.Spec.Name,
			Version: c.Config.Version.Index,
		}
	}

	data, _ := json.Marshal(states)
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash)
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
