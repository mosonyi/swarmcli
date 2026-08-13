// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/muesli/termenv"
	"github.com/stretchr/testify/require"
)

// trueColour makes a test's assertions run against styled strings. The default
// profile under `go test` is Ascii, where every style renders as plain text —
// so a width computed with len() instead of lipgloss.Width would pass here and
// break on a real terminal.
func trueColour(t *testing.T) {
	t.Helper()
	prev := lipgloss.ColorProfile()
	lipgloss.SetColorProfile(termenv.TrueColor)
	t.Cleanup(func() { lipgloss.SetColorProfile(prev) })
}

func sampleToggles() []Toggle {
	return []Toggle{
		{Label: "Autoscroll", Value: "On", Tone: ToggleOn},
		{Label: "Wrap", Value: "Off", Tone: ToggleOff},
		{Label: "Node", Value: "worker-1", Tone: ToggleInfo},
		{Label: "Stopped", Value: "Hidden", Tone: ToggleOn},
	}
}

func TestToggleRow_CentersTheBlock(t *testing.T) {
	trueColour(t)

	const width = 100
	stripped := ansi.Strip(ToggleRow(sampleToggles(), width))

	// The items keep their own gap and the slack stays outside them: on a wide
	// terminal a spread row puts the first and last item on opposite edges.
	require.Regexp(t,
		`^ +Autoscroll:On {3}Wrap:Off {3}Node:worker-1 {3}Stopped:Hidden$`,
		stripped)

	left := len(stripped) - len(strings.TrimLeft(stripped, " "))
	right := width - len(stripped)
	require.LessOrEqual(t, abs(left-right), 1, "block is centred: %d left, %d right", left, right)
}

func TestToggleRow_TightensGapsBeforeDropping(t *testing.T) {
	trueColour(t)

	// 48 columns of items: too narrow for the full gap, wide enough to keep
	// every item at a tighter one. An item dropped here would be information
	// lost to spacing.
	stripped := ansi.Strip(ToggleRow(sampleToggles(), 52))

	require.NotContains(t, stripped, "…")
	require.Regexp(t,
		`^ *Autoscroll:On +Wrap:Off +Node:worker-1 +Stopped:Hidden$`,
		stripped)
}

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

func TestToggleRow_IsAlwaysOneLine(t *testing.T) {
	trueColour(t)

	// A frame header that grows a second line silently shrinks the content the
	// frame draws — see ui.ContentRows.
	for width := 1; width <= 200; width++ {
		row := ToggleRow(sampleToggles(), width)
		require.NotContains(t, row, "\n", "width %d", width)
		require.LessOrEqual(t, lipgloss.Width(row), width, "width %d", width)
	}
}

func TestToggleRow_DropsWholeItemsWhenNarrow(t *testing.T) {
	trueColour(t)

	row := ansi.Strip(ToggleRow(sampleToggles(), 30))

	require.True(t, strings.HasSuffix(row, "…"), "the cut is marked: %q", row)
	require.Contains(t, row, "Autoscroll:On")
	require.NotContains(t, row, "Stopped", "a dropped item leaves no fragment behind")
	require.NotContains(t, row, "Node")
}

func TestToggleRow_TruncatesASingleOverlongItem(t *testing.T) {
	trueColour(t)

	row := ToggleRow([]Toggle{{Label: "Node", Value: strings.Repeat("x", 40), Tone: ToggleInfo}}, 12)

	require.Equal(t, 12, lipgloss.Width(row))
	require.True(t, strings.HasSuffix(ansi.Strip(row), "…"))
}

func TestToggleRow_Empty(t *testing.T) {
	require.Equal(t, "", ToggleRow(nil, 80))
}

func TestToggleRow_TonesAreDistinct(t *testing.T) {
	trueColour(t)

	row := ToggleRow(sampleToggles(), 100)

	// Labels share the frame title's label colour; each tone renders its value
	// in its own colour, so on/off is legible without reading the words.
	require.Contains(t, row, titleLabelStyle.Render("Autoscroll:"))
	require.Contains(t, row, toggleOnStyle.Render("On"))
	require.Contains(t, row, toggleOffStyle.Render("Off"))
	require.Contains(t, row, toggleInfoStyle.Render("worker-1"))

	require.NotEqual(t, toggleOnStyle.Render("x"), toggleOffStyle.Render("x"))
	require.NotEqual(t, toggleOnStyle.Render("x"), toggleInfoStyle.Render("x"))
	require.NotEqual(t, toggleOffStyle.Render("x"), toggleInfoStyle.Render("x"))
}
