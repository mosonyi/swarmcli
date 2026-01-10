package confirmdialog

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ResultMsg struct{ Confirmed bool }

type Model struct {
	Visible   bool
	Message   string
	Width     int
	Height    int
	ErrorMode bool // If true, shows "Close" instead of "Yes/No"
}

func New(width, height int) *Model { return &Model{Width: width, Height: height} }

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.Visible {
			return nil
		}
		if m.ErrorMode {
			// In error mode, any key closes the dialog
			switch msg.String() {
			case "enter", "esc", " ":
				m.Visible = false
				return func() tea.Msg { return ResultMsg{Confirmed: false} }
			}
		} else {
			// In confirm mode, y/n keys
			switch msg.String() {
			case "y", "Y":
				m.Visible = false
				return func() tea.Msg { return ResultMsg{Confirmed: true} }
			case "n", "N", "esc":
				return func() tea.Msg { return ResultMsg{Confirmed: false} }
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

	// Styled title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("208")). // Orange for warning
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

	// Border style
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("208")).
		Width(contentWidth + 2)

	// Build content
	var lines []string
	if m.ErrorMode {
		lines = append(lines, titleStyle.Render(" Error "))
	} else {
		lines = append(lines, titleStyle.Render(" Confirm Action "))
	}
	lines = append(lines, messageStyle.Render(m.Message))

	var helpText string
	if m.ErrorMode {
		helpText = fmt.Sprintf("%s Close", keyStyle.Render("<Enter/Esc>"))
	} else {
		helpText = fmt.Sprintf("%s Yes • %s No",
			keyStyle.Render("<y>"),
			keyStyle.Render("<n/Esc>"))
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
