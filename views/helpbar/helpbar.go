package helpbar

import (
	"github.com/charmbracelet/lipgloss"
	"strings"
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

	const colWidth = 18
	const numCols = 3

	// Break help items into column-first layout
	numRows := (len(allHelp) + numCols - 1) / numCols
	columns := make([][]string, numCols)

	for i, item := range allHelp {
		col := i / numRows
		columns[col] = append(columns[col], item)
	}

	// Render each column
	var renderedCols []string
	for _, col := range columns {
		var colLines []string
		for _, key := range col {
			colLines = append(colLines, styles.HelpStyle.Render("["+key+"]"))
		}
		renderedCols = append(renderedCols,
			lipgloss.NewStyle().
				Width(colWidth).
				Render(strings.Join(colLines, "\n")))
	}

	helpBlock := lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...)

	// Align to right of systemInfo
	infoWidth := lipgloss.Width(systemInfo)
	helpWidth := m.width - infoWidth
	if helpWidth < 0 {
		helpWidth = 0
	}

	helpAligned := lipgloss.NewStyle().
		Width(helpWidth).
		Align(lipgloss.Left).
		Render(helpBlock)

	return lipgloss.JoinHorizontal(lipgloss.Top, systemInfo, helpAligned)
}
