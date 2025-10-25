package commandinput

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	input   textinput.Model
	visible bool
	style   lipgloss.Style
}

type SubmitMsg string

func New() Model {
	ti := textinput.New()
	ti.Prompt = ":" // visible command mode indicator
	ti.Placeholder = "docker stack ls"
	ti.CharLimit = 256
	ti.Cursor.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("212")) // pink cursor
	ti.Focus()

	style := lipgloss.NewStyle().
		Foreground(lipgloss.Color("229")). // light text
		Background(lipgloss.Color("57")).  // dark blue bg
		Padding(0, 1)

	return Model{
		input:   ti,
		style:   style,
		visible: false,
	}
}

func (m Model) Visible() bool {
	return m.visible
}

func (m *Model) Show() tea.Cmd {
	m.visible = true
	m.input.Focus()
	return textinput.Blink
}

func (m *Model) Hide() {
	m.visible = false
	m.input.Blur()
	m.input.SetValue("")
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			cmd := m.input.Value()
			m.input.SetValue("")
			m.visible = false
			m.input.Blur()
			return m, func() tea.Msg { return SubmitMsg(cmd) }

		case "esc":
			m.Hide()
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

func (m Model) View() string {
	if !m.visible {
		return ""
	}
	return m.style.Render(m.input.View())
}
