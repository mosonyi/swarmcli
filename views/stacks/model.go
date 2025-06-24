package stacksview

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"swarmcli/views/helpbar"
)

type Model struct {
	viewport viewport.Model
	Visible  bool

	nodeId        string
	stackCursor   int
	stackServices []StackService
	ready         bool
}

type StackService struct {
	StackName   string
	ServiceName string
}

// Create a new instance
func New(width, height int) Model {
	vp := viewport.New(width, height)
	vp.SetContent("")
	return Model{
		viewport: vp,
		Visible:  false,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Name() string {
	return ViewName
}

func (m Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "enter", Desc: "view logs"},
		{Key: "k/up", Desc: "scr up"},
		{Key: "j/down", Desc: "scr down"},
		{Key: "pgup", Desc: "page up"},
		{Key: "pgdown", Desc: "page down"},
		{Key: "q", Desc: "close"},
	}
}

func LoadNodeStacks(nodeID string) tea.Cmd {
	return func() tea.Msg {
		services := loadNodeStacks(nodeID)
		return Msg{NodeId: nodeID, Services: services}
	}
}
