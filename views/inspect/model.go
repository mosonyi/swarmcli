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

type Format string

const (
	FormatYAML Format = "yml"
	FormatRaw  Format = "raw"
)

type Model struct {
	viewport   viewport.Model
	Root       *Node
	Title      string
	SearchTerm string
	searchMode bool
	ready      bool
	width      int
	height     int

	Format     Format // "yml" or "raw"
	RawContent string
	ParseError string
}

func New(width, height int, format Format) *Model {
	vp := viewport.New(width, height)
	vp.SetContent("")
	return &Model{
		viewport: vp,
		width:    width,
		height:   height,
		Format:   format,
	}
}

func (m *Model) SetFormat(format Format) {
	m.Format = format

	if m.Format == FormatRaw {
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
			{Key: "enter", Desc: "Apply"},
			{Key: "esc", Desc: "Cancel"},
		}
	}
	return []helpbar.HelpEntry{
		{Key: "/", Desc: "Search"},
		{Key: "j/k", Desc: "Down/up"},
		{Key: "r", Desc: "Toggle raw"},
		{Key: "q", Desc: "Close"},
	}
}

func (m *Model) OnEnter() tea.Cmd {
	return nil
}

func (m *Model) OnExit() tea.Cmd {
	return nil
}

func ParseFormat(v any) Format {
	switch x := v.(type) {
	case Format:
		return x
	case string:
		f := Format(x)
		if f == FormatYAML || f == FormatRaw {
			return f
		}
	}
	return FormatYAML
}
