package confirmdialog

import (
	"fmt"
	"strings"

	"swarmcli/ui"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ResultMsg struct{ Confirmed bool }

type Model struct {
	Visible bool
	Message string
	Width   int
	Height  int
}

func New(width, height int) Model { return Model{Width: width, Height: height} }

func (m Model) Init() tea.Cmd { return nil }

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.Visible {
			return m, nil
		}
		switch msg.String() {
		case "y", "Y":
			m.Visible = false
			return m, func() tea.Msg { return ResultMsg{Confirmed: true} }
		case "n", "N", "esc":
			return m, func() tea.Msg { return ResultMsg{Confirmed: false} }
		}
	}
	return m, nil
}

func (m Model) View() string {
	if !m.Visible {
		return ""
	}

	lines := []string{
		fmt.Sprintf("⚠️  %s", m.Message),
		"",
		"[y] Yes   [n] No",
	}

	content := strings.Join(lines, "\n")
	box := ui.RenderFramedBox("Confirm", "", content, 0) // width=0 → minimal width

	return lipgloss.Place(m.Width, m.Height, lipgloss.Center, lipgloss.Center, box)
}
