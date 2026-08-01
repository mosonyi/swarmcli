// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package dialog

// Shared dialog keys used across views, so a handler and the help text that
// advertises it cannot drift apart.
const (
	// BrowseKey opens a file browser from a dialog with a path input. It is a
	// chord, not a printable character: a printable one cannot be reserved
	// while a text input has focus, which is what made a path containing an
	// "f" impossible to type (#525).
	BrowseKey = "ctrl+o"

	// BrowseHint marks a path input with the browse key. Caret notation keeps
	// it 11 cells — the width the dialogs are already laid out around.
	BrowseHint = "[^o Browse]"

	// BrowseHelpKey is how the browse key appears in a dialog's help line.
	BrowseHelpKey = "<ctrl+o>"
)
