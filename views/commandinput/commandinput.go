// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package commandinput

import (
	"strings"
	"swarmcli/commands"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

type Model struct {
	input       textinput.Model
	active      bool
	suggestions []string
	selected    int
	errorMsg    string
}

func New() *Model {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Prompt = "> "
	ti.CharLimit = 256
	return &Model{input: ti}
}

func (m *Model) Show() tea.Cmd {
	m.active = true
	m.input.Focus()
	m.refreshSuggestions()
	return textinput.Blink
}

func (m *Model) Hide() tea.Cmd {
	m.active = false
	m.input.Blur()
	m.input.Reset()
	m.suggestions = nil
	m.selected = 0
	return nil
}

func (m *Model) ShowError(msg string) tea.Cmd {
	m.errorMsg = msg
	m.input.Focus()
	return nil
}

func (m *Model) refreshSuggestions() {
	prefix := strings.TrimSpace(m.input.Value())
	m.suggestions = commands.Suggest(prefix)
	m.selected = 0
}

func (m *Model) Visible() bool { return m.active }
