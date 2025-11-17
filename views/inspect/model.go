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
}

type Model struct {
	viewport   viewport.Model
	Root       *Node
	Title      string
	SearchTerm string
	searchMode bool
	ready      bool
	width      int
	height     int

	Format     string // "yml" or "raw"
	RawContent string
}

func New(width, height int) *Model {
	vp := viewport.New(width, height)
	vp.SetContent("")
	return &Model{
		viewport: vp,
		width:    width,
		height:   height,
	}
}

func (m *Model) SetFormat(format string) {
	if format != "raw" {
		format = "yml"
	}
	m.Format = format

	if m.Format == "raw" {
		m.viewport.SetContent(m.RawContent)
	} else {
		m.updateViewport()
	}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Name() string { return ViewName }

func (m *Model) SetTitle(t string) { m.Title = t }

// LoadInspectItem returns a cmd that sends a Msg(title, json)
func LoadInspectItem(title, jsonStr string) tea.Cmd {
	return func() tea.Msg { return Msg{Title: title, Content: jsonStr} }
}

func (m *Model) ShortHelpItems() []helpbar.HelpEntry {
	if m.searchMode {
		return []helpbar.HelpEntry{
			{Key: "enter", Desc: "apply"},
			{Key: "esc", Desc: "cancel"},
		}
	}
	return []helpbar.HelpEntry{
		{Key: "/", Desc: "search"},
		{Key: "j/k", Desc: "down/up"},
		{Key: "q", Desc: "close"},
	}
}

func (m *Model) OnEnter() tea.Cmd {
	return nil
}

func (m *Model) OnExit() tea.Cmd {
	return nil
}
