package inspectview

import (
	"swarmcli/views/helpbar"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type treeNode struct {
	Key      string
	Value    any
	Children []*treeNode
	Path     string
}

type Model struct {
	viewport      viewport.Model
	Visible       bool
	searchTerm    string
	searchIndex   int
	searchMatches []int
	mode          string // "normal", "search"
	inspectRoot   *treeNode
	expanded      map[string]bool
	inspectLines  string
	ready         bool
	title         string
}

func New(width, height int) Model {
	vp := viewport.New(width, height)
	return Model{
		viewport: vp,
		mode:     "normal",
		expanded: make(map[string]bool),
	}
}

// SetTitle sets the view title
func (m *Model) SetTitle(title string) {
	m.title = title
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Name() string {
	return ViewName
}

func (m Model) ShortHelpItems() []helpbar.HelpEntry {
	if m.mode == "search" {
		return []helpbar.HelpEntry{
			{Key: "enter", Desc: "confirm"},
			{Key: "esc", Desc: "cancel"},
			{Key: "n/N", Desc: "next/prev"},
		}
	}
	return []helpbar.HelpEntry{
		{Key: "/", Desc: "search"},
		{Key: "n/N", Desc: "next/prev"},
		{Key: "q", Desc: "close"},
		{Key: "space", Desc: "expand/collapse"},
	}
}

func LoadInspectItem(lines string) tea.Cmd {
	return func() tea.Msg {
		return Msg(lines)
	}
}
