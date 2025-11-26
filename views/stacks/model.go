package stacksview

import (
	"fmt"
	"swarmcli/docker"
	"swarmcli/ui"

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
		RenderItem: func(s docker.StackEntry, selected bool) string {
			line := fmt.Sprintf("%-20s %3d", s.Name, s.ServiceCount)
			if selected {
				return ui.CursorStyle.Render(line)
			}
			return line
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
		{Key: "i/enter", Desc: "services"},
		{Key: "k/up", Desc: "scroll up"},
		{Key: "j/down", Desc: "scroll down"},
		{Key: "pgup", Desc: "page up"},
		{Key: "pgdown", Desc: "page down"},
		{Key: "q", Desc: "close"},
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
