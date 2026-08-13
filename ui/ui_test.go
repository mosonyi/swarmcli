// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

func TestTruncateANSI_WidthAware(t *testing.T) {
	tests := []struct {
		name  string
		in    string
		width int
	}{
		{"ascii", "hello world", 5},
		{"sgr colored", "\x1b[31mhello\x1b[0m world", 4},
		{"wide cjk", "你好世界ab", 3},
		{"emoji", "ab😀cd", 3},
		{"over budget", "hi", 10},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := TruncateANSI(tc.in, tc.width)
			// Never exceeds the requested cell width (lipgloss == x/ansi width model).
			require.LessOrEqual(t, lipgloss.Width(got), tc.width)
			// Never emits a dangling escape introducer.
			require.False(t, strings.HasSuffix(got, "\x1b"))
		})
	}
	require.Equal(t, "", TruncateANSI("anything", 0))
}

func TestTruncateANSIAfter_WidthAware(t *testing.T) {
	// Skipping N non-straddling cells leaves width-N cells behind.
	require.Equal(t, 6, lipgloss.Width(TruncateANSIAfter("hello world", 5))) // " world"
	require.Equal(t, 4, lipgloss.Width(TruncateANSIAfter("你好世界", 4)))        // 4 wide = 8 cells, skip 4 -> 4 cells
	require.Equal(t, "hello", TruncateANSIAfter("hello", 0))                 // skip<=0 returns input
	// Skipping past content yields nothing.
	require.Equal(t, "", TruncateANSIAfter("hi", 10))
}

func TestRenderFramedBox_TruncatesAnOverlongTitle(t *testing.T) {
	trueColour(t)

	const w = 40
	box := RenderFramedBox(strings.Repeat("Logs(a-very-long-stack/service)", 3), "", "line", "", w)

	lines := strings.Split(box, "\n")
	require.Greater(t, len(lines), 2)
	for i, line := range lines {
		require.Equal(t, w, lipgloss.Width(line), "line %d must not overhang the box", i)
	}
	require.Contains(t, lines[0], "…", "the cut is marked")
}

// Regression for issue #369: the centered overlay must keep its left border on
// the same display column for every row, even when the background log lines
// contain wide characters at differing positions. The old rune-counting
// truncation cut at the wrong display column per row, so the box zig-zagged and
// background text bled through.
func TestOverlayCentered_AlignsWithWideChars(t *testing.T) {
	const w = 40
	// Two backgrounds of equal display width (40) but different rune layouts:
	// one front-loads wide CJK chars, the other is plain ASCII. Rune-counting
	// truncation diverges between them; cell-aware truncation does not.
	wideRow := "你你你你你" + strings.Repeat(" ", w-10) // 5 wide = 10 cells + 30 spaces
	asciiRow := strings.Repeat("x", 10) + strings.Repeat(" ", w-10)
	require.Equal(t, w, lipgloss.Width(wideRow))
	require.Equal(t, w, lipgloss.Width(asciiRow))

	var base []string
	for i := 0; i < 11; i++ {
		if i%2 == 0 {
			base = append(base, wideRow)
		} else {
			base = append(base, asciiRow)
		}
	}

	overlay := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(18).
		Render("Select Node to Filter\nnode-1\nnode-2")

	out := OverlayCentered(strings.Join(base, "\n"), overlay, w, 0)

	// The left vertical border '│' must sit at one constant display column.
	var leftCols []int
	for _, line := range strings.Split(out, "\n") {
		if idx := strings.Index(line, "│"); idx >= 0 {
			leftCols = append(leftCols, lipgloss.Width(line[:idx]))
		}
	}
	require.GreaterOrEqual(t, len(leftCols), 2, "overlay border rows should be present")
	for _, c := range leftCols[1:] {
		require.Equal(t, leftCols[0], c, "dialog left border must be column-aligned on every row")
	}
}
