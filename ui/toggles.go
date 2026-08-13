// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// ToggleTone selects the colour a toggle's value carries.
type ToggleTone int

const (
	// ToggleOff dims the value: the option is not engaged.
	ToggleOff ToggleTone = iota
	// ToggleOn highlights the value: the option is engaged.
	ToggleOn
	// ToggleInfo marks a value that is not a yes/no — a node name, a query.
	ToggleInfo
)

// Toggle value styles. Labels reuse the frame title's label colour so a status
// row reads as part of the same header as the title above it.
var (
	toggleOnStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("#00ff87")).Bold(true)
	toggleOffStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	toggleInfoStyle = titleScopeStyle
)

// Toggle is one "Label:Value" item of a status row.
type Toggle struct {
	Label string
	Value string
	Tone  ToggleTone
}

// toggleGap is the space between items once they all fit. The slack is not
// handed to the gaps beyond it: on a wide terminal that pushes the items to
// opposite edges, where they read as four unrelated labels rather than as one
// status row. It matches the help bar's column gap.
const toggleGap = 3

// ToggleRow renders items as "Label:Value", gapped by toggleGap and centred in
// width, so a view's options read at a glance instead of hiding in prose. The
// result is always a single line no wider than width: the row is a frame
// header, and a header that grows a second line silently shrinks the content
// the frame draws.
//
// A narrowing terminal closes the gaps first and only then drops items, from
// the right, marking the cut with "…" rather than truncating mid-item — a value
// cut to "Hid" reads as a state rather than as missing text.
func ToggleRow(items []Toggle, width int) string {
	if len(items) == 0 {
		return ""
	}
	if width <= 0 {
		width = 80
	}

	rendered := make([]string, len(items))
	total := 0
	for i, item := range items {
		rendered[i] = renderToggle(item)
		total += lipgloss.Width(rendered[i])
	}

	// Everything fits at a one-space gap: take the widest gap width affords.
	if gaps := len(rendered) - 1; total+gaps <= width {
		gap := toggleGap
		if gaps > 0 {
			gap = min(toggleGap, (width-total)/gaps)
		}
		return centerRow(strings.Join(rendered, strings.Repeat(" ", gap)), width)
	}

	for len(rendered) > 1 {
		rendered = rendered[:len(rendered)-1]
		line := strings.Join(rendered, " ") + " …"
		if lipgloss.Width(line) <= width {
			return centerRow(line, width)
		}
	}
	return ansi.Truncate(rendered[0], width, "…")
}

// renderToggle renders one "Label:Value" item, the value coloured by its tone.
func renderToggle(item Toggle) string {
	value := toggleOffStyle
	switch item.Tone {
	case ToggleOn:
		value = toggleOnStyle
	case ToggleInfo:
		value = toggleInfoStyle
	}
	return titleLabelStyle.Render(item.Label+":") + value.Render(item.Value)
}

// centerRow pads a row so it sits in the middle of width, under the centred
// frame title it belongs to. There is no trailing padding: the frame pads every
// line it draws to its own width.
func centerRow(row string, width int) string {
	pad := (width - lipgloss.Width(row)) / 2
	if pad <= 0 {
		return row
	}
	return strings.Repeat(" ", pad) + row
}
