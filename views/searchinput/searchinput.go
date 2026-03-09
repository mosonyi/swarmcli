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
	input   textinput.Model
	active  bool
	editing bool // true = active/typing, false = passive/locked (query shown dimmed)
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
	m.editing = true
	m.input.Focus()
	return textinput.Blink
}

// Confirm locks the search input into passive mode (query visible but not editable).
func (m *Model) Confirm() {
	m.editing = false
	m.input.Blur()
}

// Resume transitions from passive back to active editing mode.
func (m *Model) Resume() tea.Cmd {
	m.editing = true
	m.input.Focus()
	return textinput.Blink
}

// Hide deactivates the search input and resets its value.
func (m *Model) Hide() {
	m.active = false
	m.editing = false
	m.input.Blur()
	m.input.Reset()
}

// Visible reports whether the search input is currently active.
func (m *Model) Visible() bool { return m.active }

// Editing reports whether the search input is in editing mode.
func (m *Model) Editing() bool { return m.editing }

// Query returns the current input value.
func (m *Model) Query() string { return m.input.Value() }
