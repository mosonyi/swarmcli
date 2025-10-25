package commandinput

import (
	"strings"
	"swarmcli/commands"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	input   textinput.Model
	visible bool
	history []string
	histPos int
}

type (
	ShowMsg   struct{}
	HideMsg   struct{}
	SubmitMsg struct{ Command string }
)

func New() Model {
	ti := textinput.New()
	ti.Placeholder = "Enter command..."
	ti.Prompt = ": "
	ti.Focus()
	ti.CharLimit = 256

	return Model{
		input:   ti,
		visible: false,
	}
}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Visible() bool {
	return m.visible
}

func (m *Model) Show() tea.Cmd {
	m.visible = true
	m.input.Focus()
	return nil
}

func (m *Model) Hide() tea.Cmd {
	m.visible = false
	m.input.Blur()
	return nil
}

func (m Model) Update(msg tea.Msg) (Model, tea.Cmd) {
	if !m.visible {
		return m, nil
	}

	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			val := m.input.Value()
			m.history = append(m.history, val)
			m.histPos = len(m.history)
			m.input.SetValue("")
			m.visible = false
			return m, func() tea.Msg { return SubmitMsg{Command: val} }

		case "esc":
			m.visible = false
			m.input.SetValue("")
			return m, nil

		case "up":
			if m.histPos > 0 {
				m.histPos--
				m.input.SetValue(m.history[m.histPos])
				m.input.CursorEnd()
			}
		case "down":
			if m.histPos < len(m.history)-1 {
				m.histPos++
				m.input.SetValue(m.history[m.histPos])
				m.input.CursorEnd()
			} else {
				m.histPos = len(m.history)
				m.input.SetValue("")
			}
		}
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) View() string {
	if !m.visible {
		return ""
	}

	style := lipgloss.NewStyle().
		Background(lipgloss.Color("#303030")).
		Foreground(lipgloss.Color("#00d7ff")).
		Padding(0, 1)

	return style.Render(m.input.View())
}

func (m *Model) Suggestions(prefix string) []string {
	var results []string
	for _, cmd := range commands.List() {
		if strings.HasPrefix(cmd.Name(), prefix) {
			results = append(results, cmd.Name())
		}
	}
	return results
}
