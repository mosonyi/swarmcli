// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// RenderViewFrame composes a complete framed view from its parts.
// It computes frame dimensions, trims/pads content to fit, and renders
// the bordered frame. When fullscreen is true, only a centered title
// line and raw content are returned (no borders).
func RenderViewFrame(title, header, content, footer string, width, height int, fullscreen bool) string {
	if fullscreen {
		titleText := FrameTitleStyle.Render(title)
		titleLine := lipgloss.NewStyle().
			Width(width).
			Align(lipgloss.Center).
			Render(titleText)
		if header != "" {
			return titleLine + "\n" + header + "\n" + content
		}
		return titleLine + "\n" + content
	}

	// Pad header, content, and footer uniformly so columns align inside the frame.
	if header != "" {
		header = LeftPadContent(header)
	}
	content = LeftPadContent(content)
	if footer != "" {
		footer = LeftPadContent(footer)
	}

	frame := ComputeFrameDimensions(width, height, width, height, header, footer)
	trimmed := TrimOrPadContentToLines(content, frame.DesiredContentLines)
	return RenderFramedBox(title, header, trimmed, footer, frame.FrameWidth)
}

// LeftPadContent prepends a single space to each line of content.
func LeftPadContent(content string) string {
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = " " + line
	}
	return strings.Join(lines, "\n")
}
