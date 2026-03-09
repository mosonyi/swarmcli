// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package searchinput

import (
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// Model is a styled search input box that renders above the main view,
// mirroring the command input (":") component.
type Model struct {
	input  textinput.Model
	active bool
}

// New creates a new search input (inactive by default).
func New() *Model {
	ti := textinput.New()
	ti.Placeholder = ""
	ti.Prompt = "/ "
	ti.CharLimit = 256
	return &Model{input: ti}
}

// Show activates the search input and returns a blink command.
func (m *Model) Show() tea.Cmd {
	m.active = true
	m.input.Focus()
	return textinput.Blink
}

// Hide deactivates the search input and resets its value.
func (m *Model) Hide() {
	m.active = false
	m.input.Blur()
	m.input.Reset()
}

// Visible reports whether the search input is currently active.
func (m *Model) Visible() bool { return m.active }

// Query returns the current input value.
func (m *Model) Query() string { return m.input.Value() }
