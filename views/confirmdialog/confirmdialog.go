package confirmdialog

import (
	"fmt"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"swarmcli/ui"
)

type ResultMsg struct {
	Confirmed bool
}

type Model struct {
	Visible bool
	Message string
	Width   int
	Height  int
}

func New(width, height int) Model {
	return Model{Width: width, Height: height}
}

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "y", "Y":
			if m.Visible {
				return Model{Visible: false}, func() tea.Msg { return ResultMsg{Confirmed: true} }
			}
		case "n", "N", "esc":
			if m.Visible {
				return Model{Visible: false}, func() tea.Msg { return ResultMsg{Confirmed: false} }
			}
		}
	}
	return m, nil
}

func (m Model) View() string {
	if !m.Visible {
		return ""
	}

	content := fmt.Sprintf("\n\n⚠️  %s\n\n[y] Yes   [n] No", m.Message)
	centered := lipgloss.Place(
		m.Width,
		m.Height-4,
		lipgloss.Center,
		lipgloss.Center,
		content,
	)

	return ui.RenderFramedBox("Confirm", "", centered, m.Width)
}
