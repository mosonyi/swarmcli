package inspectview

import (
	"swarmcli/views/helpbar"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Node struct {
	Key      string
	Raw      any
	ValueStr string
	Children []*Node
	Parent   *Node
	Expanded bool
	Matches  bool
	Depth    int
	Path     string
}

type Model struct {
	viewport    viewport.Model
	Visible     []*Node // flattened visible nodes (according to expanded/filter)
	Root        *Node
	Cursor      int
	Title       string
	SearchTerm  string
	searchMode  bool
	searchIndex int
	ready       bool
	width       int
	height      int
}

func New(width, height int) Model {
	vp := viewport.New(width, height)
	vp.SetContent("")
	return Model{
		viewport: vp,
		Title:    "",
		ready:    false,
		width:    width,
		height:   height,
	}
}

// LoadInspectItem returns a cmd that sends a Msg(title, json)
func LoadInspectItem(title, jsonStr string) tea.Cmd {
	return func() tea.Msg { return Msg{Title: title, Content: jsonStr} }
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Name() string { return ViewName }

func (m Model) ShortHelpItems() []helpbar.HelpEntry {
	if m.searchMode {
		return []helpbar.HelpEntry{
			{Key: "enter", Desc: "apply"},
			{Key: "esc", Desc: "cancel"},
		}
	}
	return []helpbar.HelpEntry{
		{Key: "/", Desc: "search"},
		{Key: "space", Desc: "toggle"},
		{Key: "← →", Desc: "collapse/expand"},
		{Key: "j/k", Desc: "down/up"},
		{Key: "n/N", Desc: "next/prev match"},
		{Key: "q", Desc: "close"},
	}
}

func (m *Model) SetTitle(t string) { m.Title = t }
