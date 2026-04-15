// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package helpbar

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type HelpEntry struct {
	Key      string
	Desc     string
	Disabled bool
}

type Model struct {
	globalHelp  []HelpEntry
	viewHelp    []HelpEntry
	width       int
	height      int
	minColWidth int
}

// EditionLabel is rendered below the logo. Override in init() to customise.
var EditionLabel = "Community Edition"

func SetEditionLabel(label string) {
	EditionLabel = label
}

const defaultMinColWidth = 20

// KeyBack is the canonical key for "go back / close this view".
const KeyBack = "esc"

// KeyQuit is the canonical key for "quit the application".
const KeyQuit = "ctrl+q"

func New(width, height int) *Model {
	return &Model{
		globalHelp:  []HelpEntry{{Key: KeyQuit, Desc: "quit"}, {Key: "?", Desc: "help"}},
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

func (m *Model) View(systemInfo string, hasError bool) string {
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

	disabledKeyStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	disabledDescStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

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
			keyText := "<" + entry.Key + ">"
			visibleKeyLen := lipgloss.Width(keyText)
			padding := maxKeyLen - visibleKeyLen
			var line string
			if entry.Disabled {
				line = disabledKeyStyle.Render(keyText) + strings.Repeat(" ", padding+2) + disabledDescStyle.Render(entry.Desc)
			} else {
				line = keyStyle.Render(keyText) + strings.Repeat(" ", padding+2) + entry.Desc
			}
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
	logoTop := `  ___________      ___________
 /   _____/  \    /  \_   ___ \
 \_____  \\   \/\/   /    \  \/
 /        \\        /\     \____
/_______  / \__/\__/  \________/`

	logoBottom := `        \/       \/          \/`

	logoColor := lipgloss.Color("214") // yellow by default
	if hasError {
		logoColor = lipgloss.Color("9") // red when errors exist
	}
	logoStyle := lipgloss.NewStyle().
		Foreground(logoColor).
		Bold(true)

	editionStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("75")).
		Bold(true)

	// Right-align edition label on the last line
	pad := len(logoBottom) - len(EditionLabel)
	lastLine := logoStyle.Render(logoBottom[:pad]) + editionStyle.Render(EditionLabel)
	swcLogo := logoStyle.Render(logoTop) + "\n" + lastLine

	return lipgloss.JoinHorizontal(lipgloss.Top, systemInfo, helpAligned, "  ", swcLogo)
}
