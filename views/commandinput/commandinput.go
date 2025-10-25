package commandinput

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// SubmitMsg is emitted when the user presses Enter in command mode.
type SubmitMsg struct{ Command string }

// Model represents the command input bar (like in k9s).
type Model struct {
	input    textinput.Model
	visible  bool
	history  []string
	histPos  int
	errorMsg string
}

// New creates a new command input model.
func New() Model {
	ti := textinput.New()
	ti.Prompt = ": "
	ti.CharLimit = 256
	ti.Focus() // ensures cursor state initialized properly

	return Model{
		input: ti,
	}
}

// Visible returns true if the command bar is visible.
func (m Model) Visible() bool { return m.visible }

// Show makes the command bar visible and focuses the input.
func (m *Model) Show() tea.Cmd {
	m.visible = true
	m.errorMsg = ""
	m.input.Focus()
	return textinput.Blink
}

// Hide hides the command bar and clears its state.
func (m *Model) Hide() tea.Cmd {
	m.visible = false
	m.errorMsg = ""
	m.input.Blur()
	m.input.Reset()
	return nil
}

// ShowError displays an error message (without losing focus).
func (m *Model) ShowError(msg string) tea.Cmd {
	m.errorMsg = msg
	m.visible = true
	return nil
}

// Update handles key events and manages input/history state.
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
			if val == "" {
				return m, nil
			}
			m.history = append(m.history, val)
			m.histPos = len(m.history)
			m.input.Reset()
			m.errorMsg = ""
			m.visible = false
			return m, func() tea.Msg { return SubmitMsg{Command: val} }

		case "esc":
			m.Hide()
			return m, nil

		case "up":
			if len(m.history) == 0 {
				break
			}
			if m.histPos > 0 {
				m.histPos--
			}
			m.input.SetValue(m.history[m.histPos])
			m.input.CursorEnd()

		case "down":
			if len(m.history) == 0 {
				break
			}
			if m.histPos < len(m.history)-1 {
				m.histPos++
				m.input.SetValue(m.history[m.histPos])
			} else {
				m.histPos = len(m.history)
				m.input.Reset()
			}
			m.input.CursorEnd()

		default:
			// Clear error when user edits
			if m.errorMsg != "" && (msg.Type == tea.KeyRunes || msg.Type == tea.KeyBackspace) {
				m.errorMsg = ""
			}
		}
	}

	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

// View renders the command bar and optional error message.
func (m Model) View() string {
	if !m.visible {
		return ""
	}

	cmdBarStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#303030")).
		Foreground(lipgloss.Color("#00d7ff")).
		Padding(0, 1)

	view := cmdBarStyle.Render(m.input.View())

	if m.errorMsg != "" {
		errStyle := lipgloss.NewStyle().
			Foreground(lipgloss.Color("#ff5f87")).
			Bold(true).
			Padding(0, 1)
		view += "\n" + errStyle.Render(m.errorMsg)
	}

	return view
}
