package stacksview

import (
	"strings"
	"swarmcli/docker"
	"swarmcli/views/helpbar"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"

	"swarmcli/ui/components/filterable/list"
)

type Model struct {
	List    filterlist.FilterableList[docker.StackEntry]
	Visible bool
	nodeID  string
	ready   bool
	width   int
	height  int
}

func New(width, height int) *Model {
	vp := viewport.New(width, height)
	vp.SetContent("")

	list := filterlist.FilterableList[docker.StackEntry]{
		Viewport: vp,
		// Render item will be initialized later after the column with is set
		Match: func(s docker.StackEntry, query string) bool {
			return strings.Contains(strings.ToLower(s.Name), strings.ToLower(query))
		},
	}

	return &Model{
		List:    list,
		Visible: false,
		width:   width,
		height:  height,
	}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Name() string { return ViewName }

func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "i/enter", Desc: "Services"},
		{Key: "↑/↓", Desc: "Navigate"},
		{Key: "pgup", Desc: "Page up"},
		{Key: "pgdown", Desc: "Page down"},
		{Key: "/", Desc: "Filter"},
		{Key: "q", Desc: "Close"},
	}
}

func LoadStacks(nodeID string) tea.Cmd {
	return func() tea.Msg {
		snap, err := docker.GetOrRefreshSnapshot()
		if err != nil {
			return Msg{NodeID: nodeID, Stacks: nil}
		}
		stacks := snap.ToStackEntries()
		return Msg{NodeID: nodeID, Stacks: stacks}
	}
}

func (m *Model) OnEnter() tea.Cmd { return nil }
func (m *Model) OnExit() tea.Cmd  { return nil }
