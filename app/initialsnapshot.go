package app

import (
	"swarmcli/docker"
	stacksview "swarmcli/views/stacks"
	"swarmcli/views/view"

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

// loadSnapshotAndNavigateToStacksCmd loads snapshot and then navigates to stacks view
// Used after context switch to show the stacks for the new context
func loadSnapshotAndNavigateToStacksCmd() tea.Cmd {
	return func() tea.Msg {
		_, err := docker.RefreshSnapshot()
		if err != nil {
			return snapshotLoadedMsg{Err: err}
		}
		// Navigate to stacks view after snapshot loads
		return view.NavigateToMsg{
			ViewName: stacksview.ViewName,
			Replace:  true, // Replace the loading view
		}
	}
}
