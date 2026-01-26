// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package commandinput

import tea "github.com/charmbracelet/bubbletea"

type (
	// SubmitMsg is emitted when the user presses Enter in command mode.
	SubmitMsg struct {
		Command string
		Args    []string
	}

	// Command defines metadata for a recognized command.
	Command struct {
		Name        string
		Description string
		Handler     func(args []string) tea.Msg // executed when valid
	}
)
