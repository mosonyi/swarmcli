package scaledialog

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type ResultMsg struct {
	Confirmed bool
	Replicas  uint64
}

type Model struct {
	Visible     bool
	ServiceName string
	Replicas    uint64
	Width       int
	Height      int
}

func New(width, height int) *Model {
	return &Model{Width: width, Height: height}
}

func (m *Model) Init() tea.Cmd { return nil }

func (m *Model) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if !m.Visible {
			return nil
		}
		switch msg.String() {
		case "up":
			m.Replicas++
			return nil
		case "down":
			if m.Replicas > 0 {
				m.Replicas--
			}
			return nil
		case "enter":
			m.Visible = false
			return func() tea.Msg {
				return ResultMsg{Confirmed: true, Replicas: m.Replicas}
			}
		case "esc", "n", "N":
			m.Visible = false
			return func() tea.Msg {
				return ResultMsg{Confirmed: false, Replicas: m.Replicas}
			}
		}
	}
	return nil
}

func (m *Model) View() string {
	if !m.Visible {
		return ""
	}

	// Calculate content width (minimum 65 to fit help text on one line)
	contentWidth := 65
	if w := lipgloss.Width(m.ServiceName) + 20; w > contentWidth {
		contentWidth = w
	}

	// Styled title
	titleStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("15")).
		Background(lipgloss.Color("63")). // Blue for scale
		Padding(0, 1).
		Width(contentWidth)

	// Message style
	messageStyle := lipgloss.NewStyle().
		Padding(1, 2).
		Width(contentWidth)

	// Replicas display style
	replicasStyle := lipgloss.NewStyle().
		Bold(true).
		Foreground(lipgloss.Color("10")).
		Align(lipgloss.Center).
		Width(contentWidth).
		Padding(1, 0)

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
		BorderForeground(lipgloss.Color("63")).
		Width(contentWidth + 2)

	// Build content
	var lines []string
	lines = append(lines, titleStyle.Render(" Scale Service "))
	lines = append(lines, messageStyle.Render(fmt.Sprintf("Service: %s", m.ServiceName)))
	lines = append(lines, replicasStyle.Render(fmt.Sprintf("Replicas: %d", m.Replicas)))

	helpText := fmt.Sprintf("%s Increase • %s Decrease • %s Apply • %s Cancel",
		keyStyle.Render("<↑>"),
		keyStyle.Render("<↓>"),
		keyStyle.Render("<Enter>"),
		keyStyle.Render("<Esc>"))
	lines = append(lines, helpStyle.Render(helpText))

	content := strings.Join(lines, "\n")
	return borderStyle.Render(content)
}

func (m *Model) Show(serviceName string, currentReplicas uint64) *Model {
	m.Visible = true
	m.ServiceName = serviceName
	m.Replicas = currentReplicas
	return m
}

func (m *Model) Hide() *Model {
	m.Visible = false
	return m
}
