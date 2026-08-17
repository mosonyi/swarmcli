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

// oneFlexCols is a second shape for the width invariants: a single elastic
// column, so a rule that only holds when the slack divides evenly is caught.
func oneFlexCols() []Column[row] {
	return []Column[row]{
		{Label: "NAME", MinWidth: 4, Flex: true, Cell: func(r row) string { return r.a }},
		{Label: "KIND", MinWidth: 4, Cell: func(r row) string { return r.b }},
		{Label: "LABELS", MinWidth: 6, Cell: func(r row) string { return r.c }},
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

// Scrolling right stops where the text does.
//
// The offset used to climb for as long as the key was pressed, and once it
// passed the end of the string every flex cell on the row rendered blank —
// which reads as the data disappearing rather than as the end of it, and takes
// as many presses to undo as it took to cause.
func TestScrollRightStopsAtTheEndOfTheText(t *testing.T) {
	const name = "a-name-far-wider-than-its-column"
	f := FilterableList[row]{
		Columns:  testCols(),
		Filtered: []row{{a: name, b: "kind", c: "x=1"}},
		Header:   &HeaderConfig{Columns: ColumnDefs(testCols())},
	}
	f.SetOuterSize(40, 10)

	for range 50 {
		f.ScrollRight()
	}

	// Column 0 carries the leading space as well as the gap.
	nameWidth := f.ColWidths()[0] - ColGap - 1
	require.Equal(t, len(name)-nameWidth, f.columnScroll,
		"the offset stops with the last window ending at the end of the text")
	require.Contains(t, f.RenderRow(f.Filtered[0], true), "its-column",
		"so the tail is on screen rather than scrolled past")
}

// Every column shares a wide terminal's leftover equally, so the table spans the
// width and no gap in the row is wider than any other.
//
// Two narrower rules came first and each opened a hole where the surplus landed:
// on the Flex columns, which put one in the middle of every row, and on a single
// trailing column, which put the whole screen's worth at the right-hand end.
func TestLayoutWidths_EveryColumnSharesTheLeftoverEqually(t *testing.T) {
	items := []row{{a: "short", b: "kind", c: "x=1"}}

	narrow := LayoutWidths(oneFlexCols(), items, 60, -1)
	wide := LayoutWidths(oneFlexCols(), items, 200, -1)

	require.Equal(t, 200, sum(wide), "the row spans the width")
	// The non-elastic columns grow too, and by the same amount as the elastic one
	// — the rounding remainder is deliberately parked on the last column, where
	// it is trailing margin rather than one gap wider than its neighbours.
	grown := wide[0] - narrow[0]
	require.Positive(t, grown)
	require.Equal(t, grown, wide[1]-narrow[1], "a non-flex column takes the same share")
	require.InDelta(t, grown, wide[2]-narrow[2], float64(len(wide)),
		"and the last differs only by the rounding remainder parked on it")
}

// NaturalWidth must be the width at which LayoutWidths neither stretches nor
// squeezes, or a view asking "have I room to spare?" gets a wrong answer.
func TestNaturalWidth_IsTheNoStretchNoSqueezeWidth(t *testing.T) {
	items := []row{{a: "some-name", b: "kind", c: "team=platform"}}

	for _, cols := range [][]Column[row]{testCols(), oneFlexCols()} {
		natural := NaturalWidth(cols, items, -1)
		require.Equal(t, natural, sum(LayoutWidths(cols, items, natural, -1)),
			"at the natural width the layout is an identity")

		// One cell narrower and something must give; one wider and the slack
		// has to land somewhere.
		require.Equal(t, natural-1, sum(LayoutWidths(cols, items, natural-1, -1)))
		require.Equal(t, natural+10, sum(LayoutWidths(cols, items, natural+10, -1)))
	}
}

// The sort indicator widens its column, so a view must get the same answer the
// layout will act on.
func TestNaturalWidth_AccountsForTheSortIndicator(t *testing.T) {
	items := []row{{a: "x", b: "y", c: "z"}}
	require.Greater(t, NaturalWidth(testCols(), items, 0), NaturalWidth(testCols(), items, -1))
}
