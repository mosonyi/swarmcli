package stacksview

import (
	"swarmcli/docker"
	"swarmcli/views/helpbar"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	viewport viewport.Model
	Visible  bool

	nodeID  string
	cursor  int
	entries []docker.StackEntry

	ready bool
}

func New(width, height int) Model {
	vp := viewport.New(width, height)
	vp.SetContent("")
	return Model{
		viewport: vp,
		Visible:  false,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Name() string { return ViewName }

func (m Model) ShortHelpItems() []helpbar.HelpEntry {
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
