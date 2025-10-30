package inspectview

import (
	"swarmcli/views/helpbar"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var keyStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("33")) // blueish

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
	matches    []*Node
}

func New(width, height int) Model {
	vp := viewport.New(width, height)
	vp.SetContent("")
	return Model{
		viewport: vp,
		width:    width,
		height:   height,
	}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Name() string { return ViewName }

func (m *Model) SetTitle(t string) { m.Title = t }

// LoadInspectItem returns a cmd that sends a Msg(title, json)
func LoadInspectItem(title, jsonStr string) tea.Cmd {
	return func() tea.Msg { return Msg{Title: title, Content: jsonStr} }
}

func (m Model) ShortHelpItems() []helpbar.HelpEntry {
	if m.searchMode {
		return []helpbar.HelpEntry{
			{Key: "enter", Desc: "apply"},
			{Key: "esc", Desc: "cancel"},
		}
	}
	return []helpbar.HelpEntry{
		{Key: "/", Desc: "search"},
		{Key: "j/k", Desc: "down/up"},
		{Key: "n/N", Desc: "next/prev match"},
		{Key: "q", Desc: "close"},
	}
}
