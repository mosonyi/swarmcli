package commandinput

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type Model struct {
	input    textinput.Model
	visible  bool
	history  []string
	histPos  int
	errorMsg string
}

type SubmitMsg struct{ Command string }

func New() Model {
	ti := textinput.New()
	ti.Placeholder = "Enter command..."
	ti.Prompt = ": "
	ti.CharLimit = 256

	return Model{
		input:   ti,
		visible: false,
	}
}

func (m Model) Visible() bool { return m.visible }

func (m *Model) Show() tea.Cmd {
	m.visible = true
	m.input.Focus()
	m.errorMsg = ""
	return nil
}

func (m *Model) Hide() tea.Cmd {
	m.visible = false
	m.input.Blur()
	m.input.SetValue("")
	m.errorMsg = ""
	return nil
}

// ShowError sets an error message on the command input. It returns nil (no cmd),
// but keeping the signature tee-friendly allows you to return a cmd later if desired.
func (m *Model) ShowError(msg string) tea.Cmd {
	m.errorMsg = msg
	// ensure the input is visible so user sees the error
	m.visible = true
	// do not focus/clear input so user can edit immediately
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
			m.errorMsg = ""
			return m, func() tea.Msg { return SubmitMsg{Command: val} }

		case "esc":
			m.visible = false
			m.input.SetValue("")
			m.errorMsg = ""
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
		default:
			// Clear error as soon as the user starts typing so they can correct.
			if m.errorMsg != "" && (msg.Type == tea.KeyRunes || msg.Type == tea.KeyBackspace) {
				m.errorMsg = ""
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

	out := style.Render(m.input.View())

	if m.errorMsg != "" {
		errStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff5f87")).
			Bold(true).
			MarginTop(0).
			Padding(0, 1)
		out = out + "\n" + errStyle.Render(m.errorMsg)
	}

	return out
}
