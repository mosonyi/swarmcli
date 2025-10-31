package loadingview

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"swarmcli/views/helpbar"
	"swarmcli/views/view"
)

const ViewName = "loading"

type Model struct {
	Width, Height int
	message       string
	style         lipgloss.Style
}

func New(width, height int, message string) Model {
	return Model{
		Width:   width,
		Height:  height,
		message: message,
		style:   lipgloss.NewStyle().Bold(true).Align(lipgloss.Center).Foreground(lipgloss.Color("205")),
	}
}

func (m Model) Name() string {
	return ViewName
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (view.View, tea.Cmd) {
	return m, nil
}

func (m Model) ShortHelpItems() []helpbar.HelpEntry {
	return []helpbar.HelpEntry{
		{Key: "q", Desc: "close"},
	}
}

func (m Model) View() string {
	text := m.style.Render(m.message)
	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, text)
}
