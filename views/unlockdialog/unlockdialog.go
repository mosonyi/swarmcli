// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

// Package unlockdialog provides an app-level modal for entering a Docker Swarm
// unlock key. It is shown when the active context's swarm is locked.
package unlockdialog

import (
	"strings"

	"github.com/Eldara-Tech/swarmcli/ui/dialog"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

// ResultMsg is emitted when the dialog is dismissed. Confirmed is true when the
// user submitted a key with Enter; Key holds the entered (unmasked) value.
type ResultMsg struct {
	Confirmed bool
	Key       string
}

type Model struct {
	Visible bool
	Width   int
	Height  int
	input   textinput.Model
}

func New(width, height int) *Model {
	in := textinput.New()
	in.Placeholder = "SWMKEY-1-..."
	in.Prompt = "Key: "
	in.EchoMode = textinput.EchoPassword
	in.CharLimit = 256
	in.Width = 50
	return &Model{Width: width, Height: height, input: in}
}

func (m *Model) Init() tea.Cmd { return textinput.Blink }

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	if !m.Visible {
		return nil
	}
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "enter":
			m.Visible = false
			val := m.input.Value()
			return func() tea.Msg { return ResultMsg{Confirmed: true, Key: val} }
		case "esc":
			m.Visible = false
			return func() tea.Msg { return ResultMsg{Confirmed: false} }
		}
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return cmd
}

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}

	contentWidth := 65

	titleStyle := dialog.TitleStyle.Width(contentWidth)
	messageStyle := dialog.MessageStyle.Width(contentWidth)
	inputStyle := dialog.ItemStyle.Width(contentWidth)
	helpStyle := dialog.HelpStyle.Width(contentWidth)
	borderStyle := dialog.BorderStyle.Width(contentWidth + dialog.BoxInsetColumns)

	var lines []string
	lines = append(lines, titleStyle.Render(" Unlock Swarm "))
	lines = append(lines, messageStyle.Render("Enter the swarm unlock key to decrypt this cluster."))
	lines = append(lines, inputStyle.Render(m.input.View()))

	helpText := dialog.KeyStyle.Render("<Enter>") + " Unlock • " + dialog.KeyStyle.Render("<Esc>") + " Cancel"
	lines = append(lines, helpStyle.Render(helpText))

	return borderStyle.Render(strings.Join(lines, "\n"))
}

// Show makes the dialog visible with a cleared, focused input.
func (m *Model) Show() *Model {
	m.input.SetValue("")
	m.input.Focus()
	m.Visible = true
	return m
}

// Hide makes the dialog invisible and blurs the input.
func (m *Model) Hide() *Model {
	m.input.Blur()
	m.Visible = false
	return m
}
