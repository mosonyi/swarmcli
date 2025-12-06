package helpbar

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
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

func New(width, height int) *Model {
	return &Model{
		globalHelp:  []HelpEntry{{Key: "q", Desc: "quit"}, {Key: "?", Desc: "help"}},
		width:       width,
		height:      height,
		minColWidth: defaultMinColWidth,
	}
}

func (m *Model) WithGlobalHelp(entries []HelpEntry) *Model {
	m.globalHelp = entries
	return m
}

func (m *Model) WithViewHelp(entries []HelpEntry) *Model {
	m.viewHelp = entries
	return m
}

func (m *Model) SetWidth(width int) *Model {
	m.width = width
	return m
}

func (m *Model) SetHeight(height int) *Model {
	m.height = height
	return m
}

func (m *Model) SetMinColWidth(width int) *Model {
	m.minColWidth = width
	return m
}

func (m *Model) View(systemInfo string) string {
	allHelp := append(m.globalHelp, m.viewHelp...)
	if len(allHelp) == 0 {
		return systemInfo
	}

	// Reserve space for logo
	logoWidth := 32 // Increased to give more room for the logo
	infoWidth := lipgloss.Width(systemInfo)
	availableWidth := m.width - infoWidth - logoWidth
	if availableWidth < m.minColWidth {
		// Not enough space to render help, just return systemInfo
		return systemInfo
	}

	// Fixed: 5 rows per column
	rowsPerColumn := 5

	// Calculate how many columns we need
	numCols := (len(allHelp) + rowsPerColumn - 1) / rowsPerColumn

	// Check if we have space for all columns
	maxCols := availableWidth / m.minColWidth
	if maxCols < 1 {
		maxCols = 1
	}
	if numCols > maxCols {
		numCols = maxCols
	}

	// Prepare columns filled top-to-bottom
	columns := make([][]HelpEntry, numCols)

	for i, entry := range allHelp {
		col := i / rowsPerColumn
		if col >= numCols {
			// Skip items that don't fit
			break
		}
		columns[col] = append(columns[col], entry)
	}

	// Render columns with table formatting
	keyStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("39")).
		Bold(true)

	var renderedCols []string
	for colIdx, col := range columns {
		// Find max key length in this column for alignment (visible length)
		maxKeyLen := 0
		for _, entry := range col {
			keyText := "<" + entry.Key + ">"
			keyLen := lipgloss.Width(keyText)
			if keyLen > maxKeyLen {
				maxKeyLen = keyLen
			}
		}

		var lines []string
		for _, entry := range col {
			styledKey := keyStyle.Render("<" + entry.Key + ">")
			// Calculate visible padding needed using lipgloss.Width for proper Unicode handling
			keyText := "<" + entry.Key + ">"
			visibleKeyLen := lipgloss.Width(keyText)
			padding := maxKeyLen - visibleKeyLen
			line := styledKey + strings.Repeat(" ", padding+2) + entry.Desc
			lines = append(lines, line)
		}

		colContent := strings.Join(lines, "\n")

		// Add spacing between columns (3 spaces)
		if colIdx > 0 {
			renderedCols = append(renderedCols, "   ")
		}

		colBlock := lipgloss.NewStyle().
			Render(colContent)
		renderedCols = append(renderedCols, colBlock)
	}

	helpBlock := lipgloss.JoinHorizontal(lipgloss.Top, renderedCols...)

	helpAligned := lipgloss.NewStyle().
		Width(availableWidth).
		Align(lipgloss.Left).
		Render(helpBlock)

	// Add SWC logo on the right side
	logo := `   ___________      ___________  
 /   _____/  \    /  \_   ___ \ 
 \_____  \\   \/\/   /    \  \/ 
 /        \\        /\     \____
/_______  / \__/\  /  \______  /
        \/       \/          \/`

	logoStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("214")).
		Bold(true)

	swcLogo := logoStyle.Render(logo)

	return lipgloss.JoinHorizontal(lipgloss.Top, systemInfo, helpAligned, "  ", swcLogo)
}
