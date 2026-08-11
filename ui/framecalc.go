// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package ui

import "strings"

const (
	horizontalPadding = 4

	// FramedChromeRows is what RenderViewFrame's bordered layout spends on the
	// frame itself: the top border, which carries the title, and the bottom one.
	FramedChromeRows = 2
	// FullscreenChromeRows is what its borderless layout spends instead: a
	// single centered title line.
	FullscreenChromeRows = 1
)

// FrameSpec captures the calculated dimensions for a framed view.
type FrameSpec struct {
	FrameWidth          int
	FrameHeight         int
	DesiredContentLines int
}

// ComputeFrameDimensions derives consistent frame sizing across views.
//
// Inputs:
// - viewportWidth/Height: usable dimensions provided by app/update.go
// - fallbackWidth/Height: model dimensions to use if the viewport is not ready
// - header/footer: rendered strings used to count occupied lines
//
// Behavior aligns with stacks view: add 4 columns for frame padding, use the
// already-adjusted viewport height directly, and compute the inner content
// lines as frameHeight - vertical padding - header - footer (never negative).
func ComputeFrameDimensions(viewportWidth, viewportHeight, fallbackWidth, fallbackHeight int, header, footer string) FrameSpec {
	frameWidth := viewportWidth
	if frameWidth <= 0 {
		frameWidth = fallbackWidth
	}
	if frameWidth <= 0 {
		frameWidth = 80
	}
	frameWidth += horizontalPadding

	frameHeight := viewportHeight
	if frameHeight <= 0 {
		frameHeight = fallbackHeight
	}
	if frameHeight <= 0 {
		frameHeight = 20
	}

	return FrameSpec{
		FrameWidth:          frameWidth,
		FrameHeight:         frameHeight,
		DesiredContentLines: ContentRows(frameHeight, FramedChromeRows, header, footer),
	}
}

// ContentRows is the space left for content inside a frame of frameHeight rows
// once the frame's own chrome (chromeRows) and the header and footer have taken
// theirs. A view that renders a viewport should size that viewport to this, so
// what the viewport scrolls is exactly what the frame draws.
func ContentRows(frameHeight, chromeRows int, header, footer string) int {
	rows := frameHeight - chromeRows - countRows(header) - countRows(footer)
	if rows < 0 {
		rows = 0
	}
	return rows
}

func countRows(s string) int {
	if s == "" {
		return 0
	}
	return len(strings.Split(s, "\n"))
}
