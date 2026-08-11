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

// ToggleRow renders items as "Label:Value" spread evenly across width, so a
// view's options read at a glance instead of hiding in prose. The result is
// always a single line no wider than width: the row is a frame header, and a
// header that grows a second line silently shrinks the content the frame draws.
//
// When the items do not fit they are dropped from the right and the cut is
// marked with "…", rather than truncated mid-item — a value cut to "Hid" reads
// as a state rather than as missing text.
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

	// Everything fits: hand the slack to the gaps between items.
	if total+len(rendered)-1 <= width {
		return spreadRow(rendered, width, total)
	}

	for len(rendered) > 1 {
		rendered = rendered[:len(rendered)-1]
		line := strings.Join(rendered, " ") + " …"
		if lipgloss.Width(line) <= width {
			return line
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

// spreadRow distributes width-total spaces across the gaps between items, the
// remainder going to the leftmost gaps, so the row ends flush with the right
// edge. total is the summed width of items.
func spreadRow(items []string, width, total int) string {
	gaps := len(items) - 1
	if gaps == 0 {
		return items[0]
	}

	slack := width - total
	var b strings.Builder
	for i, item := range items {
		b.WriteString(item)
		if i == gaps {
			break
		}
		gap := slack / gaps
		if i < slack%gaps {
			gap++
		}
		b.WriteString(strings.Repeat(" ", gap))
	}
	return b.String()
}
