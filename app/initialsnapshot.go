package app

import (
	"swarmcli/docker"

	tea "github.com/charmbracelet/bubbletea"
)

// --- Async snapshot loader ---
func loadSnapshotAsync() tea.Cmd {
	return func() tea.Msg {
		_, err := docker.RefreshSnapshot()
		return snapshotLoadedMsg{Err: err}
	}
}

type snapshotLoadedMsg struct{ Err error }
