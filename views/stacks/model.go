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

	nodeId        string
	stackCursor   int
	stackServices []docker.StackService
	ready         bool
}

func New(width, height int) Model {
	vp := viewport.New(width, height)
	vp.SetContent("")
	return Model{viewport: vp, Visible: false}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Name() string { return ViewName }

func (m Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "enter", Desc: "view logs"},
		{Key: "r", Desc: "refresh stacks"},
		{Key: "k/up", Desc: "scr up"},
		{Key: "j/down", Desc: "scr down"},
		{Key: "pgup", Desc: "page up"},
		{Key: "pgdown", Desc: "page down"},
		{Key: "q", Desc: "close"},
	}
}

func LoadStacks(nodeID string) tea.Cmd {
	return func() tea.Msg {
		services := docker.GetStacks(nodeID)
		return Msg{NodeId: nodeID, Services: services}
	}
}
