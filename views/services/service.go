package servicesview

import (
	"context"
	"swarmcli/docker"

	tea "github.com/charmbracelet/bubbletea"
)

// serviceProgressMsg wraps a Docker progress update
type serviceProgressMsg struct {
	Progress docker.ProgressUpdate
}

func restartServiceWithProgressCmd(serviceName string, msgCh chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		l().Debugln("[CMD] Starting restart with progress for %s", serviceName)
		progressCh := make(chan docker.ProgressUpdate, 10)

		go func() {
			defer close(progressCh)

			l().Debugf("[Goroutine] Calling RestartServiceWithProgress ...")
			err := docker.RestartServiceWithProgress(context.Background(), serviceName, progressCh)
			if err != nil {
				l().Debugf("[Goroutine] RestartServiceWithProgress failed: %v", err)
			}
			l().Debugf("[Goroutine] RestartServiceWithProgress returned")
		}()

		go func() {
			defer close(msgCh)
			l().Debugf("[Listener] Starting progress listener loop")
			for progress := range progressCh {
				l().Debugf("[Listener] Got update: %d/%d", progress.Replaced, progress.Total)
				if progress.Replaced == progress.Total && progress.Total > 0 {
					sendMsg(msgCh, serviceProgressMsg{progress})
					l().Debugf("[Listener] Final update, exiting early")
					return // stop listening immediately
				}
				sendMsg(msgCh, serviceProgressMsg{progress})
			}
			l().Debugf("[Listener] Progress listener loop exiting")
		}()

		return nil
	}
}

func refreshServicesCmd(nodeID, stackName string, filterType FilterType) tea.Cmd {
	return func() tea.Msg {
		_, err := docker.RefreshSnapshot()
		if err != nil {
			return nil
		}

		entries, title := LoadServicesForView(filterType, nodeID, stackName)
		return Msg{
			Title:      title,
			Entries:    entries,
			FilterType: filterType,
			NodeID:     nodeID,
			StackName:  stackName,
		}
	}
}

func LoadServicesForView(filterType FilterType, nodeID, stackName string) (entries []ServiceEntry, title string) {
	switch filterType {
	case NodeFilter:
		entries = LoadNodeServices(nodeID)
		title = "Services on Node: " + nodeID
	case StackFilter:
		entries = LoadStackServices(stackName)
		title = "Services in Stack: " + stackName
	default: // All services
		entries = LoadStackServices("")
		title = "All Services"
	}
	return
}
