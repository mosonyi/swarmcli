package nodesview

import (
	"strings"
	"swarmcli/docker"
	filterlist "swarmcli/ui/components/filterable/list"
	"swarmcli/views/helpbar"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	List    filterlist.FilterableList[docker.NodeEntry]
	Visible bool
	ready   bool
	width   int
	height  int
}

func New(width, height int) *Model {
	vp := viewport.New(width, height)
	vp.SetContent("")

	list := filterlist.FilterableList[docker.NodeEntry]{
		Viewport: vp,
		Match: func(n docker.NodeEntry, query string) bool {
			return strings.Contains(strings.ToLower(n.Hostname), strings.ToLower(query))
		},
	}

	return &Model{
		List:    list,
		Visible: false,
		width:   width,
		height:  height,
	}
}

func (m *Model) Init() tea.Cmd {
	return nil
}

func (m *Model) Name() string {
	return ViewName
}

func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "i", Desc: "inspect"},
		{Key: "p", Desc: "ps"},
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

func (m *Model) OnEnter() tea.Cmd {
	return nil
}

func (m *Model) OnExit() tea.Cmd {
	return nil
}
