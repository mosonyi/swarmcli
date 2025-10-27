package commandinput

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Model represents the command input bar (like in k9s).
type Model struct {
	input     textinput.Model
	visible   bool
	history   []string
	histPos   int
	errorMsg  string
	commands  map[string]Command
	cmdPrefix string // usually ":" for Vim-style
}

// New creates a new command input model with a given prefix (e.g. ":").
func New(prefix string) Model {
	ti := textinput.New()
	ti.Prompt = prefix + " "
	ti.CharLimit = 256
	ti.Focus()

	return Model{
		input:     ti,
		history:   make([]string, 0),
		commands:  make(map[string]Command),
		cmdPrefix: prefix,
	}
}

// Register adds a new command definition.
func (m *Model) Register(cmd Command) {
	m.commands[cmd.Name] = cmd
}

// Visible reports whether the input bar is shown.
func (m Model) Visible() bool { return m.visible }

func (m *Model) Show() tea.Cmd {
	m.visible = true
	m.errorMsg = ""
	m.input.Focus()
	return textinput.Blink
}

func (m *Model) Hide() tea.Cmd {
	m.visible = false
	m.errorMsg = ""
	m.input.Blur()
	m.input.Reset()
	return nil
}

func (m *Model) ShowError(msg string) tea.Cmd {
	m.errorMsg = msg
	m.visible = true
	m.input.Focus()
	return nil
}
