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

// TestRenderFramedBoxBordersEveryHeaderLine — the header was appended as a
// single box line however many lines it had, so a two-line header put the
// opening border on its first row and the closing one on its last.
func TestRenderFramedBoxBordersEveryHeaderLine(t *testing.T) {
	out := RenderFramedBox("Title", "NAME      STATE\nfiltered: web", "alpha\nbravo", "1-5 of 5", 40)

	rows := strings.Split(out, "\n")
	require.Greater(t, len(rows), 2)
	for i, row := range rows[1 : len(rows)-1] {
		require.Truef(t, strings.HasPrefix(row, "│"), "row %d does not open with a border: %q", i, row)
		require.Truef(t, strings.HasSuffix(row, "│"), "row %d does not close with a border: %q", i, row)
		require.Equalf(t, 40, lipgloss.Width(row), "row %d is not the frame's width: %q", i, row)
	}
}

// TestRenderFramedBoxHeightFillsRequestedHeight — the pad loop recomputed the
// gap it was closing while closing it, so it stopped around half way and the
// box came out short of the height it was asked for.
func TestRenderFramedBoxHeightFillsRequestedHeight(t *testing.T) {
	for _, tc := range []struct {
		name           string
		header, footer string
		frameHeight    int
	}{
		{"no header or footer", "", "", 12},
		{"header", "NAME      STATE", "", 12},
		{"header and footer", "NAME      STATE", "1-5 of 5", 12},
		{"two-line header", "NAME      STATE\nfiltered: web", "1-5 of 5", 12},
		{"two-line header, tall frame", "NAME      STATE\nfiltered: web", "1-5 of 5", 24},
	} {
		t.Run(tc.name, func(t *testing.T) {
			out := RenderFramedBoxHeight("Title", tc.header, "alpha\nbravo", tc.footer, 40, tc.frameHeight)
			require.Len(t, strings.Split(out, "\n"), tc.frameHeight)
		})
	}
}

// TestRenderFramedBoxHeightCannotShrinkBelowItsChrome — when the borders,
// header and footer already fill the requested height there is nothing left to
// give, and the box is as short as it can be rather than the height asked for.
func TestRenderFramedBoxHeightCannotShrinkBelowItsChrome(t *testing.T) {
	out := RenderFramedBoxHeight("Title", "one\ntwo\nthree", "footer\nsecond", "x", 40, 6)
	require.Greater(t, len(strings.Split(out, "\n")), 6)
}

func TestBoldANSI_ReAssertsPastEveryReset(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{
			name: "plain text is wrapped once",
			in:   "hello",
			want: "\x1b[1mhello\x1b[0m",
		},
		{
			// The log view's own node prefix: a colour, then a reset, then the
			// message. Without the re-assertion the message would not be bold.
			name: "a colour ending in a reset",
			in:   "\x1b[38;5;117mweb.task@node\x1b[0m | msg",
			want: "\x1b[1m\x1b[38;5;117mweb.task@node\x1b[0m\x1b[1m | msg\x1b[0m",
		},
		{
			name: "a reset folded into a composite SGR",
			in:   "a\x1b[0;32mb",
			want: "\x1b[1ma\x1b[0;32m\x1b[1mb\x1b[0m",
		},
		{
			name: "the empty-parameter reset shorthand",
			in:   "a\x1b[mb",
			want: "\x1b[1ma\x1b[m\x1b[1mb\x1b[0m",
		},
		{
			name: "a colour that does not reset is left alone",
			in:   "a\x1b[32mb",
			want: "\x1b[1ma\x1b[32mb\x1b[0m",
		},
		{
			name: "empty in, empty out",
			in:   "",
			want: "",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Equal(t, tc.want, BoldANSI(tc.in))
		})
	}
}

func TestBoldANSI_PreservesWidthAndText(t *testing.T) {
	in := "\x1b[38;5;117mweb\x1b[0m | msg"
	out := BoldANSI(in)
	require.Equal(t, lipgloss.Width(in), lipgloss.Width(out), "bold adds no printable cells")
	require.Equal(t, "web | msg", stripSGR(out))
}

func TestBoldANSI_LeavesAnUnterminatedEscapeAlone(t *testing.T) {
	// A truncated escape at the end of a line must not send the scan past the
	// end of the string.
	require.Equal(t, "\x1b[1mabc\x1b[38;5\x1b[0m", BoldANSI("abc\x1b[38;5"))
}

// stripSGR removes CSI sequences so a test can assert on the text alone.
func stripSGR(s string) string {
	var b strings.Builder
	for i := 0; i < len(s); {
		if seq, ok := csiAt(s, i); ok {
			i += len(seq)
			continue
		}
		b.WriteByte(s[i])
		i++
	}
	return b.String()
}
