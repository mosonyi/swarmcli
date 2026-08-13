// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/require"
)

// TestRenderViewFrame_OccupiesExactlyHeight — the borderless fullscreen layout
// used to emit whatever the view handed it, so a view that had held rows back
// for borders never drawn left a dead band at the bottom of the terminal.
func TestRenderViewFrame_OccupiesExactlyHeight(t *testing.T) {
	long := strings.TrimSuffix(strings.Repeat("content\n", 100), "\n")

	cases := []struct {
		name           string
		header, footer string
		content        string
		fullscreen     bool
	}{
		{name: "fullscreen, header and footer", header: "hdr", footer: "ftr", content: long, fullscreen: true},
		{name: "fullscreen, neither", content: long, fullscreen: true},
		{name: "fullscreen, two-line footer", header: "hdr", footer: "a\nb", content: long, fullscreen: true},
		{name: "fullscreen, content shorter than the frame", header: "hdr", content: "one\ntwo", fullscreen: true},
		{name: "framed, header and footer", header: "hdr", footer: "ftr", content: long},
		{name: "framed, content shorter than the frame", header: "hdr", content: "one\ntwo"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			for _, height := range []int{5, 24, 40} {
				out := RenderViewFrame("Title", tc.header, tc.content, tc.footer, 60, height, tc.fullscreen)
				require.Equal(t, height, lipgloss.Height(out), "height=%d", height)
			}
		})
	}
}

// TestRenderViewFrame_LongTitleKeepsTheHeight — the fullscreen title line is
// centred in a fixed width, which wraps an over-long title onto further rows
// and pushes the frame past the height it was given.
func TestRenderViewFrame_LongTitleKeepsTheHeight(t *testing.T) {
	title := strings.Repeat("Logs(a-long-stack/a-long-service)", 3)

	fullscreen := RenderViewFrame(title, "hdr", "body", "ftr", 40, 8, true)
	require.Equal(t, 8, lipgloss.Height(fullscreen))
	for i, row := range strings.Split(fullscreen, "\n") {
		require.LessOrEqual(t, lipgloss.Width(row), 40, "fullscreen row %d", i)
	}

	// The bordered layout draws to width+4; every row of the box, the title's
	// top border included, has to be that same width.
	framed := RenderViewFrame(title, "hdr", "body", "ftr", 40, 8, false)
	require.Equal(t, 8, lipgloss.Height(framed))
	rows := strings.Split(framed, "\n")
	bottom := lipgloss.Width(rows[len(rows)-1])
	for i, row := range rows {
		require.Equal(t, bottom, lipgloss.Width(row), "framed row %d", i)
	}
}

// TestRenderViewFrame_FullscreenKeepsHeaderAndFooter — fullscreen dropped the
// footer while the caller had already reserved its rows.
func TestRenderViewFrame_FullscreenKeepsHeaderAndFooter(t *testing.T) {
	out := RenderViewFrame("Title", "the-header", "body", "the-footer", 60, 10, true)
	rows := strings.Split(out, "\n")

	require.Contains(t, rows[0], "Title")
	require.Contains(t, rows[1], "the-header")
	require.Contains(t, rows[len(rows)-1], "the-footer")
}

func TestContentRows_NeverNegative(t *testing.T) {
	require.Equal(t, 0, ContentRows(1, FramedChromeRows, "hdr", "ftr"))
	require.Equal(t, 0, ContentRows(-5, FullscreenChromeRows, "", ""))
}
