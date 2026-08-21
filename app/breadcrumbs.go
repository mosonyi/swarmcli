// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package app

import (
	"fmt"
	"github.com/Eldara-Tech/swarmcli/v2/ui"
	"github.com/Eldara-Tech/swarmcli/v2/views/view"

	"github.com/charmbracelet/lipgloss"
)

// RenderBreadcrumbs produces the breadcrumb bar string from a list of view names.
// Top-level views show only their own name. Nested views are trimmed to the
// nearest top-level ancestor, then capped at maxDisplay items (with "…" prefix).
func RenderBreadcrumbs(names []string, maxDisplay int) string {
	if len(names) == 0 {
		return ""
	}
	// Current view is top-level → show just that view
	current := names[len(names)-1]
	if view.IsTopLevel(current) {
		style := ui.Rainbow[0]
		return style.Render(fmt.Sprintf(" %s ", current))
	}

	// Walk backward to find nearest top-level ancestor; slice from there
	start := 0
	for i := len(names) - 1; i >= 0; i-- {
		if view.IsTopLevel(names[i]) {
			start = i
			break
		}
	}
	visible := names[start:]

	// Cap at maxDisplay, prepend ellipsis if trimmed
	ellipsis := false
	if len(visible) > maxDisplay {
		visible = visible[len(visible)-maxDisplay:]
		ellipsis = true
	}

	var parts []string
	if ellipsis {
		parts = append(parts, lipgloss.NewStyle().Faint(true).Render(" … "))
	}
	for i, name := range visible {
		if i > 0 || ellipsis {
			parts = append(parts, lipgloss.NewStyle().Faint(true).Render(" → "))
		}
		style := ui.Rainbow[i%len(ui.Rainbow)]
		parts = append(parts, style.Render(fmt.Sprintf(" %s ", name)))
	}

	return lipgloss.JoinHorizontal(lipgloss.Left, parts...)
}
