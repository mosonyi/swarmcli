// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package dialog

import "github.com/charmbracelet/lipgloss"

// Shared dialog styles used across views for consistent dialog rendering.
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("63")).
			Padding(0, 1)

	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("117"))

	ItemStyle = lipgloss.NewStyle().
			Padding(0, 1)

	SelectedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("15")).
			Background(lipgloss.Color("63")).
			Padding(0, 1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1)

	KeyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("63")).
			Bold(true)
)
