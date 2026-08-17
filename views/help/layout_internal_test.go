// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package helpview

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

func TestColumnCount(t *testing.T) {
	require.Equal(t, 1, columnCount(60, 6))
	require.Equal(t, 1, columnCount(minCategoryWidth*2-1, 6))
	require.Equal(t, 2, columnCount(minCategoryWidth*2, 6))
	require.Equal(t, 4, columnCount(200, 6))
	require.Equal(t, 3, columnCount(200, 3), "never more columns than categories")
	require.Equal(t, 1, columnCount(0, 3), "a zero width still renders")
	require.Equal(t, 1, columnCount(200, 0))
}

func TestPackColumns_KeepsOrderAndTheColumnBudget(t *testing.T) {
	blocks := []string{"a", "bb\nbb", "c", "ddd\nddd\nddd", "e", "f"}

	for _, cols := range []int{0, 1, 2, 3, 4, 6, 9} {
		packed := packColumns(blocks, cols)
		require.LessOrEqual(t, len(packed), max(cols, 1))
		require.Equal(t, strings.Join(blocks, "\n\n"), strings.Join(packed, "\n\n"),
			"cols=%d reordered or dropped a category", cols)
	}
}

// TestPackColumns_IsTheBestSplitAvailable checks the claim the binary search
// makes, against every contiguous split there is. The greedy fill it replaced
// fails this: it meets the average and then empties the remainder into the last
// column.
func TestPackColumns_IsTheBestSplitAvailable(t *testing.T) {
	blocks := []string{"a", "bb\nbb", "c\nc\nc\nc\nc", "ddd\nddd\nddd", "e", "f\nf"}

	for _, cols := range []int{2, 3, 4, 5} {
		packed := packColumns(blocks, cols)
		require.Equal(t, bestTallest(blocks, cols), tallest(packed),
			"cols=%d is not the most even split available", cols)
	}
}

func TestPackColumns_ABlockTallerThanTheCeilingStillFits(t *testing.T) {
	blocks := []string{"a", strings.Repeat("x\n", 40), "b"}

	packed := packColumns(blocks, 2)
	require.LessOrEqual(t, len(packed), 2)
	require.Equal(t, strings.Join(blocks, "\n\n"), strings.Join(packed, "\n\n"))
}

func tallest(columns []string) int {
	h := 0
	for _, c := range columns {
		if n := lipgloss.Height(c); n > h {
			h = n
		}
	}
	return h
}

// bestTallest is the smallest achievable tallest-column height over every
// contiguous split of blocks into at most cols columns. Exhaustive, which is
// affordable because the input is a handful of categories.
func bestTallest(blocks []string, cols int) int {
	best := -1
	var walk func(start, columnsLeft int, worst int)
	walk = func(start, columnsLeft, worst int) {
		if start == len(blocks) {
			if best < 0 || worst < best {
				best = worst
			}
			return
		}
		if columnsLeft == 0 {
			return
		}
		for end := start + 1; end <= len(blocks); end++ {
			// A column's height is its blocks plus the blank line between them,
			// which is how packColumns joins and measures them.
			h := lipgloss.Height(strings.Join(blocks[start:end], "\n\n"))
			walk(end, columnsLeft-1, max(worst, h))
		}
	}
	walk(0, cols, 0)
	return best
}
