package helpbar

import (
	"github.com/charmbracelet/lipgloss"
	"strings"
	"swarmcli/styles"
)

type Model struct {
	globalHelp  []string
	viewHelp    []string
	width       int
	minColWidth int
}

const defaultMinColWidth = 18

func New(width int) Model {
	return Model{
		globalHelp:  []string{"q: quit", "?: help"},
		width:       width,
		minColWidth: defaultMinColWidth, // Minimum width for each help column
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

func (m Model) SetMinColWidth(width int) Model {
	m.minColWidth = width
	return m
}

func (m Model) View(systemInfo string) string {
	// Return systemInfo alone if no help
	if len(m.globalHelp) == 0 && len(m.viewHelp) == 0 {
		return systemInfo
	}

	itemStyle := styles.HelpStyle.Padding(0).Margin(0)

	// Calculate available width after systemInfo
	infoWidth := lipgloss.Width(systemInfo)
	availableWidth := m.width - infoWidth
	if availableWidth < 10 {
		return systemInfo // no room to display help
	}

	// Min col width and number of columns
	minColWidth := m.minColWidth
	maxCols := availableWidth / minColWidth
	if maxCols < 1 {
		maxCols = 1
	}

	// Reserve first column for globalHelp
	// Remaining columns for viewHelp
	viewHelpCount := len(m.viewHelp)
	numViewCols := maxCols - 1
	if numViewCols < 1 {
		// no room for viewHelp columns, just show globalHelp
		numViewCols = 0
	}

	// Determine rows for viewHelp columns
	viewRows := 0
	if numViewCols > 0 {
		viewRows = (viewHelpCount + numViewCols - 1) / numViewCols
	}

	// Build columns slice: first column = globalHelp
	columns := make([][]string, maxCols)
	columns[0] = m.globalHelp

	// Distribute viewHelp into columns 1..N, column-first top-down
	for i, item := range m.viewHelp {
		col := 1 + (i / viewRows)
		row := i % viewRows

		// Ensure enough rows in column
		for len(columns[col]) <= row {
			columns[col] = append(columns[col], "")
		}
		columns[col][row] = item
	}

	// Render columns with fixed width
	var renderedCols []string
	for _, col := range columns {
		var lines []string
		for _, key := range col {
			if key == "" {
				lines = append(lines, "") // empty line for alignment
			} else {
				lines = append(lines, itemStyle.Render("["+key+"]"))
			}
		}
		colBlock := lipgloss.NewStyle().
			Width(minColWidth).
			Padding(0).
			Margin(0).
			Render(strings.Join(lines, "\n"))
		renderedCols = append(renderedCols, colBlock)
	}

	helpBlock := lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...)

	// Right-align help block next to systemInfo
	helpAligned := lipgloss.NewStyle().
		Width(availableWidth).
		Align(lipgloss.Left).
		Padding(0).
		Margin(0).
		Render(helpBlock)

	return lipgloss.JoinHorizontal(lipgloss.Top, systemInfo, helpAligned)
}
