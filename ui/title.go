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
