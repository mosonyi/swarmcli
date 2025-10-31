package nodesview

import (
	"swarmcli/docker"
	"swarmcli/views/helpbar"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/docker/docker/api/types/swarm"
)

type Model struct {
	viewport viewport.Model
	Visible  bool

	entries []docker.NodeEntry
	cursor  int

	ready bool
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
		{Key: "s", Desc: "select"},
		{Key: "i", Desc: "inspect"},
		{Key: "k/up", Desc: "scr up"},
		{Key: "j/down", Desc: "scr down"},
		{Key: "q", Desc: "close"},
	}
}

func LoadNodes() []docker.NodeEntry {
	snapshot := docker.GetSnapshot()
	return snapshot.ToNodeEntries()
}

func LoadNodesCmd() tea.Cmd {
	return func() tea.Msg {
		entries := LoadNodes()
		return Msg{Entries: entries}
	}
}

func (m *Model) SelectedNode() *swarm.Node {
	snap := docker.GetSnapshot()
	if len(m.entries) == 0 {
		return nil
	}
	selected := m.entries[m.cursor]
	for _, n := range snap.Nodes {
		if n.ID == selected.ID {
			return &n
		}
	}
	return nil
}
