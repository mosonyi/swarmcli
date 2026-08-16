// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package dialog

import "github.com/charmbracelet/lipgloss"

// The dialog palette. Each colour is named for the role it plays instead of
// being repeated as a literal at every call site, so what a dialog is saying
// with a colour is readable, and there is one place to change it.
const (
	// An accent tints both a dialog's title bar and its border; it is what
	// tells one kind of dialog from another at a glance, and the only thing a
	// dialog should need to override. The view frame underneath has its own
	// colour, ui.FrameBorderColor, which no dialog borrows — a modal that
	// draws its frame in the frame's own colour stops reading as a layer.
	AccentPrimary = lipgloss.Color("63")  // ordinary dialogs, selections, info
	AccentWarning = lipgloss.Color("208") // confirmations that destroy something
	AccentError   = lipgloss.Color("196") // errors
	AccentEdit    = lipgloss.Color("214") // label add and remove

	BrightFg = lipgloss.Color("15")  // text on an accent, and focused input
	MutedFg  = lipgloss.Color("250") // unselected rows
	HelpFg   = lipgloss.Color("240") // the key hints along the bottom
)

// Shared dialog styles used across views for consistent dialog rendering.
var (
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(BrightFg).
			Background(AccentPrimary).
			Padding(0, 1)

	BorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(AccentPrimary)

	ItemStyle = lipgloss.NewStyle().
			Padding(0, 1)

	// MessageStyle is for a block of prose rather than a row — the sentence a
	// confirmation asks its question in. It is inset further than a row, and
	// stands off the title above it.
	MessageStyle = lipgloss.NewStyle().
			Padding(1, 2)

	SelectedStyle = lipgloss.NewStyle().
			Foreground(BrightFg).
			Background(AccentPrimary).
			Padding(0, 1)

	HelpStyle = lipgloss.NewStyle().
			Foreground(HelpFg).
			Padding(0, 1)

	KeyStyle = lipgloss.NewStyle().
			Foreground(AccentPrimary).
			Bold(true)
)
