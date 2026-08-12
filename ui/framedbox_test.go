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
