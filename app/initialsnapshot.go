package app

import (
	"swarmcli/docker"

	tea "github.com/charmbracelet/bubbletea"
)

func loadInitialSnapshot() tea.Cmd {
	return func() tea.Msg {
		_, err := docker.RefreshSnapshot()
		if err != nil {
			return snapshotErrorMsg{Err: err}
		}
		return snapshotLoadedMsg{}
	}
}

type snapshotLoadedMsg struct{}
type snapshotErrorMsg struct{ Err error }
