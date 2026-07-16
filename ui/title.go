// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package ui

import (
	"fmt"

	"github.com/charmbracelet/lipgloss"
)

// Frame-title token styles. A scoped title reads "Label(scope)[count]"; the
// label and its surrounding separators share one colour so the title reads as
// a single unit, while the scope and count are highlighted distinctly.
var (
	titleLabelStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#8be4e4")).Bold(true)
	titleScopeStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff04ff")).Bold(true)
	titleCountStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#ffefd5")).Bold(true)
)

// ScopedTitle renders a frame title of the form "Label(scope)[count]" with the
// label/separators, scope, and count coloured distinctly. The returned string
// already carries its own ANSI styling, so RenderFramedBox passes it through
// unchanged instead of re-applying FrameTitleStyle (see styleFrameTitle).
func ScopedTitle(label, scope string, count int) string {
	return titleLabelStyle.Render(label) +
		titleLabelStyle.Render("(") +
		titleScopeStyle.Render(scope) +
		titleLabelStyle.Render(")") +
		titleLabelStyle.Render("[") +
		titleCountStyle.Render(fmt.Sprintf("%d", count)) +
		titleLabelStyle.Render("]")
}

// maxFilterFragmentRunes caps the filter text baked into a title so a long
// query cannot overflow the frame title.
const maxFilterFragmentRunes = 24

// FilterFragment renders a trailing " </query>" fragment marking an active `/`
// filter, mirroring k9s (which appends the filter rather than folding it into
// the scope). The angle brackets share the label colour while the "/query"
// reuses the scope colour, so the filter reads as a scope refinement. It
// returns "" when no filter is active. Like the rest of ScopedTitle the result
// already carries ANSI styling, so styleFrameTitle passes it through unchanged.
func FilterFragment(query string) string {
	if query == "" {
		return ""
	}
	if r := []rune(query); len(r) > maxFilterFragmentRunes {
		query = string(r[:maxFilterFragmentRunes-1]) + "…"
	}
	return " " + titleLabelStyle.Render("<") +
		titleScopeStyle.Render("/"+query) +
		titleLabelStyle.Render(">")
}

// ScopedTitleFiltered renders ScopedTitle and, when filter is non-empty, appends
// a FilterFragment so the header reflects the active `/` filter, e.g.
// "Stacks(all)[1] </pos>". The count is expected to be the post-filter row
// count. See FilterFragment.
func ScopedTitleFiltered(label, scope string, count int, filter string) string {
	return ScopedTitle(label, scope, count) + FilterFragment(filter)
}
