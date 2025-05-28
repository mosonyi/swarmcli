package inspectview

import (
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	viewport      viewport.Model
	Visible       bool
	searchTerm    string
	searchIndex   int
	searchMatches []int  // indexes of match positions
	mode          string // "normal", "search"
	inspectLines  string
	ready         bool
}

func New(width, height int) Model {
	vp := viewport.New(width, height)
	return Model{
		viewport: vp,
		mode:     "normal",
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}
