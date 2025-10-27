package helpview

import (
	"swarmcli/views/helpbar"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	viewport viewport.Model
	width    int
	height   int
}

func New(width, height int) Model {
	m := Model{
		width:  width,
		height: height,
	}
	m.viewport = viewport.New(width, height)
	m.viewport.SetContent(m.buildContent())
	return m
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Name() string {
	return ViewName
}

func (m Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "q", Desc: "close"},
	}
}

func Load() tea.Cmd { return nil }
