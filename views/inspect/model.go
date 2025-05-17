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
}

func New(width, height int) Model {
	vp := viewport.New(width, height)
	return Model{
		viewport: vp,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) SetSize(width, height int) Model {
	m.viewport.Width = width
	m.viewport.Height = height - 4 // adjust for borders or header
	return m
}
