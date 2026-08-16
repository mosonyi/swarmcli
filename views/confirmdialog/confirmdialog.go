// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package confirmdialog

import (
	"fmt"
	"strings"

	"github.com/Eldara-Tech/swarmcli/ui/dialog"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/reflow/wordwrap"
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
			// Info mode with a checkbox: Space toggles it instead of closing,
			// so a notice can carry a "do not show again" opt-out. Enter/Esc
			// still close and report the checkbox state.
			if m.InfoMode && m.CheckboxLabel != "" && msg.String() == " " {
				m.CheckboxChecked = !m.CheckboxChecked
				return nil
			}
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

	// Cap dialog width to a sensible maximum, then word-wrap.
	// 100 keeps long single-token content (URLs, paths) on one line on
	// modern terminals; the m.Width-6 clamp below scales it down on
	// narrow ones.
	maxWidth := 100
	if m.Width > 0 && m.Width-6 < maxWidth {
		maxWidth = m.Width - 6
	}
	if maxWidth < 40 {
		maxWidth = 40
	}

	wrappedMessage := wordwrap.String(m.Message, maxWidth-4)

	// Content width = longest wrapped line + padding.
	contentWidth := 0
	for _, line := range strings.Split(wrappedMessage, "\n") {
		if w := lipgloss.Width(line); w > contentWidth {
			contentWidth = w
		}
	}
	contentWidth += 4
	if contentWidth < 50 {
		contentWidth = 50
	}
	if contentWidth > maxWidth {
		contentWidth = maxWidth
	}

	// The accent varies by mode, and tints the title and the border alike.
	accent := dialog.AccentWarning
	if m.InfoMode {
		accent = dialog.AccentPrimary
	}
	titleStyle := dialog.TitleStyle.Background(accent).Width(contentWidth)
	messageStyle := dialog.MessageStyle.Width(contentWidth)
	helpStyle := dialog.HelpStyle.Width(contentWidth)
	borderStyle := dialog.BorderStyle.BorderForeground(accent).Width(contentWidth + dialog.BoxInsetColumns)

	// Build content
	var lines []string
	if m.InfoMode {
		lines = append(lines, titleStyle.Render(" Info "))
	} else if m.ErrorMode {
		lines = append(lines, titleStyle.Render(" Error "))
	} else {
		lines = append(lines, titleStyle.Render(" Confirm Action "))
	}
	lines = append(lines, messageStyle.Render(wrappedMessage))

	// Add checkbox if label is provided (confirm or info mode; never error mode)
	if m.CheckboxLabel != "" && !m.ErrorMode {
		checkboxStyle := dialog.ItemStyle.Width(contentWidth)

		checkMark := "[ ]"
		if m.CheckboxChecked {
			checkMark = "[✓]"
		}
		checkboxText := fmt.Sprintf("%s %s",
			dialog.KeyStyle.Render(checkMark),
			m.CheckboxLabel)
		lines = append(lines, checkboxStyle.Render(checkboxText))
	}

	var helpText string
	switch {
	case m.InfoMode && m.CheckboxLabel != "":
		helpText = fmt.Sprintf("%s Toggle • %s Close",
			dialog.KeyStyle.Render("<Space>"),
			dialog.KeyStyle.Render("<Enter/Esc>"))
	case m.ErrorMode || m.InfoMode:
		helpText = fmt.Sprintf("%s Close", dialog.KeyStyle.Render("<Enter/Esc>"))
	case m.CheckboxLabel != "":
		helpText = fmt.Sprintf("%s Yes • %s No • %s Toggle",
			dialog.KeyStyle.Render("<y>"),
			dialog.KeyStyle.Render("<n/Esc>"),
			dialog.KeyStyle.Render("<Space>"))
	default:
		helpText = fmt.Sprintf("%s Yes • %s No",
			dialog.KeyStyle.Render("<y>"),
			dialog.KeyStyle.Render("<n/Esc>"))
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
