package servicesview

import (
	"context"
	"swarmcli/docker"
	"swarmcli/utils/log"
	"time"

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

func LoadServicesForView(filterType FilterType, nodeID, stackName string) (entries []docker.ServiceEntry, title string) {
	switch filterType {
	case NodeFilter:
		entries = docker.LoadNodeServices(nodeID)
		title = "Services on Node: " + nodeID
	case StackFilter:
		entries = docker.LoadStackServices(stackName)
		title = "Services in Stack: " + stackName
	default: // All services
		entries = docker.LoadStackServices("")
		title = "All Services"
	}
	return
}

// CheckServicesCmd checks if services have changed and returns update message if so
func CheckServicesCmd(lastHash string, filterType FilterType, nodeID, stackName string) tea.Cmd {
	return func() tea.Msg {
		logger := swarmlog.L()
		logger.Info("CheckServicesCmd: Polling for service changes")
		
		_, err := docker.RefreshSnapshot()
		if err != nil {
			logger.Errorf("CheckServicesCmd: RefreshSnapshot failed: %v", err)
			// Continue with cached snapshot
		}
		
		entries, title := LoadServicesForView(filterType, nodeID, stackName)
		newHash := computeServicesHash(entries)
		
		logger.Infof("CheckServicesCmd: lastHash=%s, newHash=%s, serviceCount=%d", 
			lastHash[:8], newHash[:8], len(entries))
		
		// Only return update message if something changed
		if newHash != lastHash {
			logger.Info("CheckServicesCmd: Change detected! Refreshing service list")
			return Msg{
				Title:      title,
				Entries:    entries,
				FilterType: filterType,
				NodeID:     nodeID,
				StackName:  stackName,
			}
		}
		
		logger.Info("CheckServicesCmd: No changes detected, scheduling next poll")
		// Schedule next poll in 5 seconds
		return tea.Tick(PollInterval, func(t time.Time) tea.Msg {
			return TickMsg(t)
		})()
	}
}
