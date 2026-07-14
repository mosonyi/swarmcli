// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package view

import (
	"swarmcli/views/helpbar"

	tea "github.com/charmbracelet/bubbletea"
)

type View interface {
	Update(msg tea.Msg) tea.Cmd
	View() string
	Init() tea.Cmd
	Name() string
	ShortHelpItems() []helpbar.HelpEntry

	// Lifecycle hooks:

	OnEnter() tea.Cmd // Called when view becomes active
	OnExit() tea.Cmd  // Called when view is removed/replaced

	HasErrors() bool // Returns true if the view has any errors to display

	// Frame components — views return unframed content,
	// app wraps with ui.RenderViewFrame.
	FrameTitle() string   // text for frame title bar
	FrameHeader() string  // rendered header line(s), "" if none
	FrameFooter() string  // rendered footer line(s), "" if none
	FrameContent() string // unframed content (may include dialog overlays)
}

// Filterable is an opt-in interface for views that support app-level "/"
// search filtering. Checked via type assertion in app/update.go.
type Filterable interface {
	ApplySearchQuery(query string)
	ClearSearchQuery()
}

// Chromeless is an opt-in interface for views that own the entire terminal.
// When Chromeless() reports true, the app renders View() verbatim — no help
// bar, no view frame, no breadcrumb bar — and hands the view the full terminal
// size instead of the chrome-reduced viewport. App-level overlays (startup,
// unlock, error, update) still composite on top. The ":" command bar and "/"
// search bar are suppressed, as there is nowhere to draw them.
//
// Two contracts, both load-bearing:
//
// The value must be constant for the view's lifetime. It selects both the
// chrome and the size handed to the view, and there is no re-resize hook on the
// render path, so flipping it mid-life leaves the view sized for chrome that is
// no longer drawn.
//
// View() must emit exactly terminalHeight lines with no trailing newline. Bubble
// Tea drops lines from the *top* when the frame is taller than the terminal, so
// a single extra line silently eats the view's first row.
type Chromeless interface {
	Chromeless() bool
}

// IsChromeless reports whether v owns the full terminal.
func IsChromeless(v View) bool {
	c, ok := v.(Chromeless)
	return ok && c.Chromeless()
}
