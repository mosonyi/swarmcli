package commandinput

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

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
