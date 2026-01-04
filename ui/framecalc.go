package ui

import "strings"

const horizontalPadding = 4

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
// lines as frameHeight - borders - header - footer (never negative).
func ComputeFrameDimensions(viewportWidth, viewportHeight, fallbackWidth, fallbackHeight int, header, footer string) FrameSpec {
	frameWidth := viewportWidth
	if frameWidth <= 0 {
		frameWidth = fallbackWidth
	}
	if frameWidth <= 0 {
		frameWidth = 80
	}
	frameWidth += 4
	frameWidth += horizontalPadding

	frameHeight := viewportHeight
	if frameHeight <= 0 {
		frameHeight = fallbackHeight
	}
	if frameHeight <= 0 {
		frameHeight = 20
	}

	headerLines := 0
	if header != "" {
		headerLines = len(strings.Split(header, "\n"))
	}
	footerLines := 0
	if footer != "" {
		footerLines = len(strings.Split(footer, "\n"))
	}

	desiredContentLines := frameHeight - 2 - headerLines - footerLines
	if desiredContentLines < 0 {
		desiredContentLines = 0
	}

	return FrameSpec{
		FrameWidth:          frameWidth,
		FrameHeight:         frameHeight,
		DesiredContentLines: desiredContentLines,
	}
}
