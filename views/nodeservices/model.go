package nodeservicesview

import (
	"swarmcli/views/helpbar"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	viewport viewport.Model
	Visible  bool

	entries []ServiceEntry
	cursor  int
	title   string
	ready   bool

	serviceColWidth int
	stackColWidth   int
	replicaColWidth int
}

type ServiceEntry struct {
	StackName      string
	ServiceName    string
	ServiceID      string
	ReplicasOnNode int
	ReplicasTotal  int
}

// Create new instance
func New(width, height int) Model {
	vp := viewport.New(width, height)
	return Model{
		viewport: vp,
		Visible:  false,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Name() string { return ViewName }

func (m Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "i", Desc: "inspect"},
		{Key: "k/up", Desc: "up"},
		{Key: "j/down", Desc: "down"},
		{Key: "q", Desc: "close"},
	}
}

func (m *Model) SetContent(msg Msg) {
	m.title = msg.Title
	m.entries = msg.Entries
	m.cursor = 0
	if m.ready {
		m.viewport.GotoTop()
		m.viewport.SetContent(m.renderEntries())
	}
}

func LoadStackServices(nodeID, nodeHostname string) tea.Cmd {
	return func() tea.Msg {
		entries := LoadEntries(nodeID)
		return Msg{
			Title:   "Services on Node: " + nodeHostname,
			Entries: entries,
		}
	}
}
