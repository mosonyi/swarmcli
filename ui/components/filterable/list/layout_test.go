// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package filterlist

import (
	"testing"

	"github.com/stretchr/testify/require"
)

type row struct {
	a, b, c string
}

func testCols() []Column[row] {
	return []Column[row]{
		{Label: "NAME", MinWidth: 4, Flex: true, Cell: func(r row) string { return r.a }},
		{Label: "KIND", MinWidth: 4, Cell: func(r row) string { return r.b }},
		{Label: "LABELS", MinWidth: 6, Flex: true, Cell: func(r row) string { return r.c }},
	}
}

func sum(xs []int) int {
	t := 0
	for _, x := range xs {
		t += x
	}
	return t
}

func TestLayoutWidths_WideFitsContent(t *testing.T) {
	cols := testCols()
	long := "this-is-a-very-long-name-value-1234567890"
	items := []row{{a: long, b: "secret", c: "team=platform"}}

	w := LayoutWidths(cols, items, 200, 0)
	require.Len(t, w, 3)
	require.LessOrEqual(t, sum(w), 200)
	// Column 0 usable content width (minus gap and leading space) fits the name.
	require.GreaterOrEqual(t, w[0]-ColGap-1, displayWidth(long))
}

func TestLayoutWidths_NarrowKeepsGapAndLabel(t *testing.T) {
	cols := testCols()
	items := []row{{
		a: "this-is-a-very-long-name-value",
		b: "kind",
		c: "team=platform,env=prod,tier=frontend",
	}}

	w := LayoutWidths(cols, items, 30, 0)
	require.Len(t, w, 3)
	// Even fully shrunk, every column keeps room for its (untruncated) label plus
	// the gap, so the header never overflows and columns never merge.
	for i, c := range cols {
		require.GreaterOrEqual(t, w[i]-ColGap, displayWidth(c.Label), "col %q", c.Label)
	}
}

func TestLayoutWidths_SortArrowFloor(t *testing.T) {
	cols := testCols()
	items := []row{{a: "x", b: "y", c: "z"}}
	// Sort on column 1 (KIND, label width 4) on a narrow terminal.
	w := LayoutWidths(cols, items, 20, 1)
	require.GreaterOrEqual(t, w[1]-ColGap, displayWidth(cols[1].Label)+2,
		"sorted column must reserve room for the ' ▲' indicator")
}

func TestLayoutWidths_EmptyAndZeroWidth(t *testing.T) {
	cols := testCols()
	require.NotPanics(t, func() {
		w := LayoutWidths(cols, nil, 0, -1) // empty items, zero width → defaults to 80
		require.Len(t, w, 3)
		for i, c := range cols {
			require.GreaterOrEqual(t, w[i]-ColGap, displayWidth(c.Label), "col %q", c.Label)
		}
	})
	require.Nil(t, LayoutWidths[row](nil, nil, 80, -1))
}

func TestRenderRow_WidthMatchesLayout(t *testing.T) {
	cols := testCols()
	items := []row{{a: "alpha", b: "kind", c: "k=v"}}
	w := LayoutWidths(cols, items, 80, 0)
	got := RenderRow(cols, w, items[0], 0, false)
	require.Equal(t, sum(w), displayWidth(got), "row width must equal sum of column widths")
}

func TestRenderRow_LengthMismatchFallback(t *testing.T) {
	cols := testCols()
	got := RenderRow(cols, []int{5}, row{a: "alpha"}, 0, false)
	require.Equal(t, "alpha", got)
}

func TestTruncateRunes_RuneAware(t *testing.T) {
	require.Equal(t, "café", TruncateRunes("café", 4))
	require.Equal(t, "ca…", TruncateRunes("café", 3))
	require.Equal(t, "…", TruncateRunes("café", 1))
	// 日本語 are wide-by-rune; ensure we slice on runes, not bytes.
	require.Equal(t, "日本…", TruncateRunes("日本語テスト", 3))
}

func TestScrollWindow_RuneAware(t *testing.T) {
	require.Equal(t, "", ScrollWindow("", 3, 10))
	// Offset slides the window; never splits a multibyte rune.
	require.Equal(t, "本語", ScrollWindow("日本語", 1, 10))
	// Offset past the end clamps to empty.
	require.Equal(t, "", ScrollWindow("abc", 99, 10))
	// Windowed content still truncates with an ellipsis when more remains.
	require.Equal(t, "bcde…", ScrollWindow("abcdefgh", 1, 5))
}

// growCols separates the two elasticities: NAME gives up width when squeezed
// and scrolls when truncated, while the trailing column takes the leftover.
func growCols() []Column[row] {
	return []Column[row]{
		{Label: "NAME", MinWidth: 4, Flex: true, Cell: func(r row) string { return r.a }},
		{Label: "KIND", MinWidth: 4, Cell: func(r row) string { return r.b }},
		{Label: "LABELS", MinWidth: 6, Flex: true, Grow: true, Cell: func(r row) string { return r.c }},
	}
}

// Slack goes to the Grow column, so the columns before it stay put however wide
// the terminal gets. A table of short values is where that matters: padding a
// middle column out opens a void between it and its neighbour.
func TestLayoutWidths_GrowAbsorbsSlackNotFlex(t *testing.T) {
	items := []row{{a: "short", b: "kind", c: "x=1"}}

	narrow := LayoutWidths(growCols(), items, 60, -1)
	wide := LayoutWidths(growCols(), items, 200, -1)

	require.Equal(t, narrow[0], wide[0], "the flex column must not absorb slack")
	require.Equal(t, narrow[1], wide[1])
	require.Greater(t, wide[2], narrow[2], "the grow column takes it instead")
	require.Equal(t, 200, sum(wide), "and the row still spans the width")
}

// A view that declares no Grow keeps the behaviour it had before Grow existed,
// so adding the field changed nothing for views that have not opted in.
func TestLayoutWidths_FlexStillAbsorbsWhenNoGrowDeclared(t *testing.T) {
	items := []row{{a: "short", b: "kind", c: "x=1"}}

	narrow := LayoutWidths(testCols(), items, 60, -1)
	wide := LayoutWidths(testCols(), items, 200, -1)

	require.Greater(t, wide[0], narrow[0], "the flex columns still share the slack")
	require.Greater(t, wide[2], narrow[2])
	require.Equal(t, 200, sum(wide))
}

// Grow is about width the table does not need; it must not rescue its column
// from shrinking when the terminal is too narrow for the content.
func TestLayoutWidths_GrowColumnStillShrinksWhenNarrow(t *testing.T) {
	items := []row{{a: "a-fairly-long-name-here", b: "kind", c: "team=platform,env=prod"}}

	w := LayoutWidths(growCols(), items, 40, -1)
	require.LessOrEqual(t, sum(w), 40)
	for i, c := range growCols() {
		require.GreaterOrEqual(t, w[i]-ColGap, displayWidth(c.Label), "column %d must still fit its label", i)
	}
}
