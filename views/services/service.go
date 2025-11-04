package servicesview

import (
	"context"
	"log"
	"swarmcli/docker"

	tea "github.com/charmbracelet/bubbletea"
)

// serviceProgressMsg wraps a Docker progress update
type serviceProgressMsg struct {
	Progress docker.ProgressUpdate
}

func restartServiceWithProgressCmd(serviceName string, msgCh chan tea.Msg) tea.Cmd {
	return func() tea.Msg {
		log.Printf("[CMD] Starting restart with progress for %s", serviceName)

		progressCh := make(chan docker.ProgressUpdate, 10)

		go func() {
			log.Println("[Goroutine] Calling RestartServiceWithProgress ...")
			err := docker.RestartServiceWithProgress(context.Background(), serviceName, progressCh)
			if err != nil {
				log.Printf("[Goroutine] RestartServiceWithProgress failed: %v", err)
			}
			log.Println("[Goroutine] RestartServiceWithProgress returned")
			close(progressCh)
		}()

		go func() {
			log.Println("[Listener] Starting progress listener loop")
			for progress := range progressCh {
				log.Printf("[Listener] Got update: %d/%d", progress.Replaced, progress.Total)
				sendMsg(msgCh, serviceProgressMsg{progress})
			}
			log.Println("[Listener] Progress listener loop exiting")
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
