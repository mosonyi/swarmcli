// SPDX-License-Identifier: Apache-2.0
// Copyright © 2026 Eldara Tech

package app

import (
	"github.com/Eldara-Tech/swarmcli/docker"

	tea "github.com/charmbracelet/bubbletea"
)

// --- Async snapshot loader ---
// loadSnapshotAsync refreshes the snapshot and reports the outcome via
// snapshotLoadedMsg, whose handler navigates to the stacks view on success
// (and surfaces a notice when the swarm is locked). Used on startup, after a
// context switch, and after an unlock.
func loadSnapshotAsync() tea.Cmd {
	return func() tea.Msg {
		_, err := docker.RefreshSnapshot()
		return snapshotLoadedMsg{Err: err}
	}
}

type snapshotLoadedMsg struct{ Err error }
