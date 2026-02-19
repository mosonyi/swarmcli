// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package filterlist

// ColumnDef describes a single column in the header.
type ColumnDef struct {
	Label    string // Base label, e.g. "STACK"
	Pct      int    // Width as % of total (0 = not percentage-based)
	MinWidth int    // Floor for column width (0 = no minimum)
	// If Pct == 0, column participates in equal distribution of remaining width.
}

// HeaderConfig holds column definitions and sort state for header rendering.
type HeaderConfig struct {
	Columns []ColumnDef

	// SortIndicator returns (columnIndex, ascending) for the current sort.
	// Return colIndex -1 to hide all arrows.
	SortIndicator func() (colIndex int, ascending bool)

	// DynamicLabel optionally overrides a column's label at render time.
	// Return "" to keep the base label. Nil means no overrides.
	DynamicLabel func(colIndex int, baseLabel string) string

	// ColWidthsFunc, when non-nil, overrides the default column width
	// computation. Receives effective width, returns widths per column.
	// Escape hatch for views with custom layout algorithms (services).
	ColWidthsFunc func(width int) []int
}

// FooterConfig controls footer rendering.
type FooterConfig struct {
	ItemLabel string // e.g. "Stack", "Secret" — used in "Stack 3 of 10"

	// Override, when non-nil, replaces the standard footer entirely.
	Override func(cursor, filteredCount int, mode ModeType, query string) string
}

// computeColWidths calculates column widths from ColumnDef definitions.
func computeColWidths(cols []ColumnDef, totalWidth int) []int {
	n := len(cols)
	if n == 0 {
		return nil
	}
	if totalWidth < 1 {
		totalWidth = 80
	}

	widths := make([]int, n)

	// Allocate percentage-based columns first
	pctUsed := 0
	var equalCols []int
	for i, c := range cols {
		if c.Pct > 0 {
			widths[i] = (totalWidth * c.Pct) / 100
			pctUsed += widths[i]
		} else {
			equalCols = append(equalCols, i)
		}
	}

	// Distribute remaining width equally among Pct==0 columns
	remaining := max(totalWidth-pctUsed, 0)
	if len(equalCols) > 0 {
		base := remaining / len(equalCols)
		rem := remaining - base*len(equalCols)
		for j, idx := range equalCols {
			widths[idx] = base
			if j < rem {
				widths[idx]++
			}
		}
	}

	// Apply MinWidth floors: steal from larger columns if needed
	for i, c := range cols {
		if c.MinWidth > 0 && widths[i] < c.MinWidth {
			deficit := c.MinWidth - widths[i]
			// Steal from earlier columns that can spare it
			for j := i - 1; j >= 0 && deficit > 0; j-- {
				floor := max(5, cols[j].MinWidth)
				spare := widths[j] - floor
				if spare <= 0 {
					continue
				}
				take := min(deficit, spare)
				widths[j] -= take
				deficit -= take
			}
			widths[i] = c.MinWidth
		}
	}

	// Floor all widths to 1
	for i := range widths {
		if widths[i] < 1 {
			widths[i] = 1
		}
	}

	return widths
}
