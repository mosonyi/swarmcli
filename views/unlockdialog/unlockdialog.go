// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

// Package unlockdialog provides an app-level modal for entering a Docker Swarm
// unlock key. It is shown when the active context's swarm is locked.
package unlockdialog

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
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

	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("63")).
		Padding(0, 1).
		Width(contentWidth)

	messageStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Width(contentWidth)

	inputStyle := lipgloss.NewStyle().
		Padding(0, 2).
		Width(contentWidth)

	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(1, 2, 0, 2).
		Width(contentWidth)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("63")).
		Bold(true)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Width(contentWidth + 2)

	var lines []string
	lines = append(lines, titleStyle.Render(" Unlock Swarm "))
	lines = append(lines, messageStyle.Render("Enter the swarm unlock key to decrypt this cluster."))
	lines = append(lines, inputStyle.Render(m.input.View()))

	helpText := keyStyle.Render("<Enter>") + " Unlock • " + keyStyle.Render("<Esc>") + " Cancel"
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
