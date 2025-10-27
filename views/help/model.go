package helpview

import (
	"swarmcli/views/helpbar"
	"swarmcli/views/view"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	viewport viewport.Model
	width    int
	height   int
}

func New(width, height int) (view.View, tea.Cmd) {
	m := Model{
		width:  width,
		height: height,
	}
	m.viewport = viewport.New(width, height)
	m.viewport.SetContent(m.buildContent())
	return m, nil
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
