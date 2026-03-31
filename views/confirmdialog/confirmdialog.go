// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package confirmdialog

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ResultMsg struct {
	Confirmed       bool
	CheckboxChecked bool // State of the optional checkbox
}

type Model struct {
	Visible         bool
	Message         string
	Width           int
	Height          int
	ErrorMode       bool   // If true, shows "Error" title and dismiss-only
	InfoMode        bool   // If true, shows "Info" title and dismiss-only
	CheckboxLabel   string // If non-empty, shows a checkbox with this label
	CheckboxChecked bool   // State of the checkbox
}

func New(width, height int) *Model { return &Model{Width: width, Height: height} }

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.Visible {
			return nil
		}
		if msg.Type == tea.KeyEsc {
			m.Visible = false
			return func() tea.Msg { return ResultMsg{Confirmed: false, CheckboxChecked: m.CheckboxChecked} }
		}
		if m.ErrorMode || m.InfoMode {
			// In error/info mode, any key closes the dialog
			switch msg.String() {
			case "enter", "esc", " ":
				m.Visible = false
				return func() tea.Msg { return ResultMsg{Confirmed: false, CheckboxChecked: m.CheckboxChecked} }
			}
		} else {
			// In confirm mode, y/n keys and space for checkbox
			switch msg.String() {
			case " ":
				// Toggle checkbox if it's available
				if m.CheckboxLabel != "" {
					m.CheckboxChecked = !m.CheckboxChecked
				}
				return nil
			case "y", "Y":
				m.Visible = false
				return func() tea.Msg { return ResultMsg{Confirmed: true, CheckboxChecked: m.CheckboxChecked} }
			case "n", "N", "esc":
				m.Visible = false
				return func() tea.Msg { return ResultMsg{Confirmed: false, CheckboxChecked: m.CheckboxChecked} }
			}
		}
	}
	return nil
}

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}

	// Calculate content width based on message
	contentWidth := lipgloss.Width(m.Message) + 4
	if contentWidth < 50 {
		contentWidth = 50
	}

	// Styled title — color varies by mode
	titleColor := lipgloss.Color("208") // Orange for warning/confirm
	if m.InfoMode {
		titleColor = lipgloss.Color("63") // Blue/purple for info
	}
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(titleColor).
		Padding(0, 1).
		Width(contentWidth)

	// Message style
	messageStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Width(contentWidth)

	// Help style
	helpStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Padding(0, 2).
		Width(contentWidth)

	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("63")).
		Bold(true)

	// Border style — matches title color
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(titleColor).
		Width(contentWidth + 2)

	// Build content
	var lines []string
	if m.InfoMode {
		lines = append(lines, titleStyle.Render(" Info "))
	} else if m.ErrorMode {
		lines = append(lines, titleStyle.Render(" Error "))
	} else {
		lines = append(lines, titleStyle.Render(" Confirm Action "))
	}
	lines = append(lines, messageStyle.Render(m.Message))

	// Add checkbox if label is provided
	if m.CheckboxLabel != "" && !m.ErrorMode && !m.InfoMode {
		checkboxStyle := lipgloss.NewStyle().
			Padding(0, 2).
			Width(contentWidth)

		checkMark := "[ ]"
		if m.CheckboxChecked {
			checkMark = "[✓]"
		}
		checkboxText := fmt.Sprintf("%s %s",
			keyStyle.Render(checkMark),
			m.CheckboxLabel)
		lines = append(lines, checkboxStyle.Render(checkboxText))
	}

	var helpText string
	if m.ErrorMode || m.InfoMode {
		helpText = fmt.Sprintf("%s Close", keyStyle.Render("<Enter/Esc>"))
	} else {
		if m.CheckboxLabel != "" {
			helpText = fmt.Sprintf("%s Yes • %s No • %s Toggle",
				keyStyle.Render("<y>"),
				keyStyle.Render("<n/Esc>"),
				keyStyle.Render("<Space>"))
		} else {
			helpText = fmt.Sprintf("%s Yes • %s No",
				keyStyle.Render("<y>"),
				keyStyle.Render("<n/Esc>"))
		}
	}
	lines = append(lines, helpStyle.Render(helpText))

	content := strings.Join(lines, "\n")
	return borderStyle.Render(content)
}

func (m *Model) WithMessage(msg string) *Model {
	m.Message = msg
	return m
}

func (m *Model) Show(msg string) *Model {
	m.Visible = true
	m.Message = msg
	return m
}

func (m *Model) Hide() *Model {
	m.Visible = false
	return m
}
