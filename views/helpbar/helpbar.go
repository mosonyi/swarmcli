package helpbar

import (
	"github.com/charmbracelet/lipgloss"
	"swarmcli/styles"
)

type Model struct {
	globalHelp []string
	viewHelp   []string
	width      int
}

func New(width int) Model {
	return Model{
		globalHelp: []string{"q: quit", "?: help"},
		width:      width,
	}
}

func (m Model) WithGlobalHelp(keys []string) Model {
	m.globalHelp = keys
	return m
}

func (m Model) WithViewHelp(keys []string) Model {
	m.viewHelp = keys
	return m
}

func (m Model) SetWidth(width int) Model {
	m.width = width
	return m
}

func (m Model) View(systemInfo string) string {
	allHelp := append(m.globalHelp, m.viewHelp...)

	// Render help columns
	const colWidth = 18
	var cols []string
	for _, key := range allHelp {
		cols = append(cols, lipgloss.NewStyle().
			Width(colWidth).
			PaddingRight(1).
			Render(styles.HelpStyle.Render("["+key+"]")))
	}
	help := lipgloss.JoinHorizontal(lipgloss.Top, cols...)

	// Layout: system info left, help right
	infoWidth := lipgloss.Width(systemInfo)
	helpWidth := m.width - infoWidth
	if helpWidth < 0 {
		helpWidth = 0
	}

	helpAligned := lipgloss.NewStyle().
		Width(helpWidth).
		Align(lipgloss.Right).
		Render(help)

	return lipgloss.JoinHorizontal(lipgloss.Top, systemInfo, helpAligned)
}
