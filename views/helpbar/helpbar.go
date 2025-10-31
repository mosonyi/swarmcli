package helpbar

import (
	"github.com/charmbracelet/lipgloss"
	"strings"
	"swarmcli/ui"
)

type HelpEntry struct {
	Key  string
	Desc string
}

type Model struct {
	globalHelp  []HelpEntry
	viewHelp    []HelpEntry
	width       int
	height      int
	minColWidth int
}

const defaultMinColWidth = 20

func New(width, height int) Model {
	return Model{
		globalHelp:  []HelpEntry{{Key: "q", Desc: "quit"}, {Key: "?", Desc: "help"}},
		width:       width,
		height:      height,
		minColWidth: defaultMinColWidth,
	}
}

func (m Model) WithGlobalHelp(entries []HelpEntry) Model {
	m.globalHelp = entries
	return m
}

func (m Model) WithViewHelp(entries []HelpEntry) Model {
	m.viewHelp = entries
	return m
}

func (m Model) SetWidth(width int) Model {
	m.width = width
	return m
}

func (m Model) SetHeight(height int) Model {
	m.height = height
	return m
}

func (m Model) SetMinColWidth(width int) Model {
	m.minColWidth = width
	return m
}

func (m Model) View(systemInfo string) string {
	allHelp := append(m.globalHelp, m.viewHelp...)
	if len(allHelp) == 0 {
		return systemInfo
	}

	itemStyle := ui.HelpStyle.Padding(0).Margin(0)

	infoWidth := lipgloss.Width(systemInfo)
	availableWidth := m.width - infoWidth
	if availableWidth < m.minColWidth {
		// Not enough space to render help, just return systemInfo
		return systemInfo
	}

	// Calculate max columns we can fit horizontally
	maxCols := availableWidth / m.minColWidth
	if maxCols < 1 {
		maxCols = 1
	}
	if maxCols > len(allHelp) {
		maxCols = len(allHelp)
	}

	// Calculate rows needed (max rows is limited by height)
	maxRows := m.height
	if maxRows < 1 {
		maxRows = 1
	}

	// Use min of rows needed and maxRows
	requiredRows := (len(allHelp) + maxCols - 1) / maxCols
	numRows := requiredRows
	if numRows > maxRows {
		numRows = maxRows
	}

	// Prepare columns filled top-to-bottom first
	columns := make([][]HelpEntry, maxCols)

	for i, entry := range allHelp {
		col := i / numRows // Fill columns top to bottom
		columns[col] = append(columns[col], entry)
	}

	// Render columns
	var renderedCols []string
	for _, col := range columns {
		var lines []string
		for _, entry := range col {
			line := itemStyle.Render(entry.Key) + "    " + entry.Desc
			lines = append(lines, line)
		}
		colBlock := lipgloss.NewStyle().
			Width(m.minColWidth).
			Render(strings.Join(lines, "\n"))
		renderedCols = append(renderedCols, colBlock)
	}

	helpBlock := lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...)

	helpAligned := lipgloss.NewStyle().
		Width(availableWidth).
		Align(lipgloss.Left).
		Render(helpBlock)

	return lipgloss.JoinHorizontal(lipgloss.Top, systemInfo, helpAligned)
}
